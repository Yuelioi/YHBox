// capture_wgc: 用 Windows Graphics Capture (WGC) + D3D11 给单个 HWND 抓帧.
//
// 暴露给 Go 端 4 个 C ABI 函数 (详见 ../README 或 pkg/capture/wgc_windows.go).
// 一个 session 对应一个 HWND; 多 bot 各开各的 session 互不干扰. DLL 内部:
//   - 全局 HashMap<sid, Arc<Mutex<Session>>>: 只在 lookup/insert/remove 持全局锁
//   - 每个 Session 自己一把 Mutex: grab 期间持 per-session 锁, 不同 sid 不互锁
//
// 帧抓取走 FramePool::try_get_next_frame() 同步轮询: caller 调 grab 时如果还
// 没新帧就返回 ERR_NOT_READY, 让 Go 那边按自己的节奏重试. 这样 DLL 这边不需要
// 给 WinRT 注册 callback (Go syscall 接 callback 会麻烦很多).

use once_cell::sync::Lazy;
use std::cell::RefCell;
use std::collections::HashMap;
use std::ffi::{c_char, c_void, CString};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};

use windows::core::{IInspectable, Interface};
use windows::Graphics::Capture::{
    Direct3D11CaptureFramePool, GraphicsCaptureItem, GraphicsCaptureSession,
};
use windows::Graphics::DirectX::DirectXPixelFormat;
use windows::Graphics::SizeInt32;
use windows::Wdk::System::SystemServices::RtlGetVersion;
use windows::Win32::Foundation::{DXGI_STATUS_OCCLUDED, HWND};
use windows::Win32::Graphics::Direct3D::D3D_DRIVER_TYPE_HARDWARE;
use windows::Win32::Graphics::Direct3D11::{
    D3D11CreateDevice, ID3D11Device, ID3D11DeviceContext, ID3D11Texture2D, D3D11_CPU_ACCESS_READ,
    D3D11_CREATE_DEVICE_BGRA_SUPPORT, D3D11_MAPPED_SUBRESOURCE, D3D11_MAP_READ, D3D11_SDK_VERSION,
    D3D11_TEXTURE2D_DESC, D3D11_USAGE_STAGING,
};
use windows::Win32::Graphics::Dxgi::{DXGI_ERROR_WAIT_TIMEOUT, IDXGIDevice};
use windows::Win32::System::SystemInformation::OSVERSIONINFOW;
use windows::Win32::System::WinRT::Direct3D11::{
    CreateDirect3D11DeviceFromDXGIDevice, IDirect3DDxgiInterfaceAccess,
};
use windows::Win32::System::WinRT::Graphics::Capture::IGraphicsCaptureItemInterop;
use windows::Win32::System::WinRT::{RoInitialize, RO_INIT_MULTITHREADED};

// 返回码约定 (i32)
pub const OK: i32 = 0;
pub const ERR_BUF_TOO_SMALL: i32 = -1;
pub const ERR_NOT_READY: i32 = -2;
pub const ERR_INVALID_SESSION: i32 = -11;
pub const ERR_GRAB_FAILED: i32 = -12;
pub const ERR_MAP_FAILED: i32 = -13;

thread_local! {
    // 每个线程一份 last error 字符串. Go 端拿到 sid==0 或负 return 后立刻读, 跟
    // 调用同线程语义 (Go syscall 期间 runtime 不会切线程).
    // 注意: 返回的指针只在下次 set_err 之前有效 — Go 端必须立即 copy, 不能保留指针.
    static LAST_ERR: RefCell<CString> = RefCell::new(CString::new("").unwrap());
}

fn set_err<S: AsRef<str>>(msg: S) {
    let s = CString::new(msg.as_ref()).unwrap_or_else(|_| CString::new("err").unwrap());
    LAST_ERR.with(|c| *c.borrow_mut() = s);
}

// DLL 进程初始化 COM MTA. WinRT (FramePool / CaptureItem / CaptureSession) 调用
// 要求线程处于 multi-threaded apartment. Go runtime 切线程后 COM 状态依赖"上次
// 谁初始化的", 不可控 — 显式 init 是廉价保险. 重复/冲突 init 返 Err 被吞.
static COM_INIT: Lazy<()> = Lazy::new(|| {
    // S_FALSE / RPC_E_CHANGED_MODE 等都吞掉, 重复 init 或上层已 init 都不阻塞.
    let _ = unsafe { RoInitialize(RO_INIT_MULTITHREADED) };
});

struct Session {
    d3d_device: ID3D11Device,
    d3d_context: ID3D11DeviceContext,
    // FramePool 内部不持有 GraphicsCaptureItem 的强引用 — 必须自己 keep alive,
    // 否则 capture session 会突然停止. 这里挂着不动就行.
    #[allow(dead_code)]
    item: GraphicsCaptureItem,
    frame_pool: Direct3D11CaptureFramePool,
    session: GraphicsCaptureSession,
    // 当前帧池配置的尺寸; 窗口被 resize 时需要 recreate frame pool.
    pool_w: i32,
    pool_h: i32,
    // staging texture cache. 仅当 staging_w/h 跟当前帧尺寸不一致时重建, 否则复用.
    // 避免每帧 CreateTexture2D 重型 GPU 资源分配 (60fps × N bot 累计 driver 压力大).
    staging: Option<ID3D11Texture2D>,
    staging_w: i32,
    staging_h: i32,
}

// 全局 session 表: lookup/insert/remove 才持锁, grab 自己走 per-session Mutex.
// 这样多 bot 同时 grab 不同 sid 不互锁 (旧版整段 grab 含 GPU copy 都串行化).
static SESSIONS: Lazy<Mutex<HashMap<u64, Arc<Mutex<Session>>>>> =
    Lazy::new(|| Mutex::new(HashMap::new()));
static NEXT_SID: AtomicU64 = AtomicU64::new(1);

// 创建 D3D11 + WinRT IDirect3DDevice, 并构造 GraphicsCaptureItem + FramePool.
fn open_session_impl(hwnd: HWND) -> windows::core::Result<Session> {
    unsafe {
        // 1. D3D11 device (BGRA support for WGC)
        let mut d3d_device: Option<ID3D11Device> = None;
        let mut d3d_context: Option<ID3D11DeviceContext> = None;
        D3D11CreateDevice(
            None,
            D3D_DRIVER_TYPE_HARDWARE,
            None,
            D3D11_CREATE_DEVICE_BGRA_SUPPORT,
            None,
            D3D11_SDK_VERSION,
            Some(&mut d3d_device),
            None,
            Some(&mut d3d_context),
        )?;
        let d3d_device = d3d_device.ok_or_else(|| {
            windows::core::Error::new(windows::core::HRESULT(-1), "D3D11 device null")
        })?;
        let d3d_context = d3d_context.ok_or_else(|| {
            windows::core::Error::new(windows::core::HRESULT(-1), "D3D11 context null")
        })?;

        // 2. 拿 DXGI device → 包成 WinRT IDirect3DDevice (给 FramePool 用)
        let dxgi: IDXGIDevice = d3d_device.cast()?;
        let inspectable: IInspectable = CreateDirect3D11DeviceFromDXGIDevice(&dxgi)?;
        let direct3d_device: windows::Graphics::DirectX::Direct3D11::IDirect3DDevice =
            inspectable.cast()?;

        // 3. HWND → GraphicsCaptureItem
        let interop: IGraphicsCaptureItemInterop =
            windows::core::factory::<GraphicsCaptureItem, IGraphicsCaptureItemInterop>()?;
        let item: GraphicsCaptureItem = interop.CreateForWindow(hwnd)?;

        // 4. FramePool + Session
        let size = item.Size()?;
        let frame_pool = Direct3D11CaptureFramePool::CreateFreeThreaded(
            &direct3d_device,
            DirectXPixelFormat::B8G8R8A8UIntNormalized,
            2, // 双缓冲
            size,
        )?;
        let session = frame_pool.CreateCaptureSession(&item)?;

        // SetIsCursorCaptureEnabled 从 Win10 1903 (build 18362) 就支持, 直接调.
        let _ = session.SetIsCursorCaptureEnabled(false);

        // SetIsBorderRequired 真正生效要 Win11 / Server 2022 (build 20348+).
        // Win10 任何版本调用都被系统忽略, 黄框无法关闭 (微软硬性 UI 提示).
        // 检查 build 号避免在不支持的系统上无谓地调.
        let mut osv = OSVERSIONINFOW {
            dwOSVersionInfoSize: std::mem::size_of::<OSVERSIONINFOW>() as u32,
            ..Default::default()
        };
        let _ = RtlGetVersion(&mut osv);
        if osv.dwBuildNumber >= 20348 {
            let _ = session.SetIsBorderRequired(false);
        }

        session.StartCapture()?;

        Ok(Session {
            d3d_device,
            d3d_context,
            item,
            frame_pool,
            session,
            pool_w: size.Width,
            pool_h: size.Height,
            staging: None,
            staging_w: 0,
            staging_h: 0,
        })
    }
}

/// 返回最近一次错误描述 (C 字符串, UTF-8). 线程本地, 每线程一份.
/// 调用方不要 free 这个指针. 指针只在下次 set_err 前有效, 必须立即 copy.
#[no_mangle]
pub extern "C" fn wgc_last_error() -> *const c_char {
    LAST_ERR.with(|c| c.borrow().as_ptr())
}

/// 为 HWND 开抓帧会话.
/// 成功: 返回非 0 sid, 把 capture surface W/H (DPI-aware 下通常 ≈ 客户区) 写入 out_w/out_h.
/// 失败: 返回 0, 错误细节看 wgc_last_error().
#[no_mangle]
pub extern "C" fn wgc_session_open(hwnd: u64, out_w: *mut i32, out_h: *mut i32) -> u64 {
    Lazy::force(&COM_INIT);
    let hwnd = HWND(hwnd as *mut c_void);
    match open_session_impl(hwnd) {
        Ok(s) => {
            unsafe {
                if !out_w.is_null() {
                    *out_w = s.pool_w;
                }
                if !out_h.is_null() {
                    *out_h = s.pool_h;
                }
            }
            let sid = NEXT_SID.fetch_add(1, Ordering::Relaxed);
            SESSIONS
                .lock()
                .unwrap()
                .insert(sid, Arc::new(Mutex::new(s)));
            sid
        }
        Err(e) => {
            set_err(format!(
                "open: {:#x} {}",
                e.code().0 as u32,
                e.message()
            ));
            0
        }
    }
}

/// 抓最新帧到 buf (BGRA, 行无 padding).
/// 返回 OK / ERR_BUF_TOO_SMALL / ERR_NOT_READY / ERR_*.
/// out_w/out_h 在 OK 和 ERR_BUF_TOO_SMALL 时都写入当前帧尺寸.
#[no_mangle]
pub extern "C" fn wgc_session_grab(
    sid: u64,
    buf: *mut u8,
    buf_len: u32,
    out_w: *mut i32,
    out_h: *mut i32,
) -> i32 {
    // 只在 lookup 时持全局锁, 拿到 Arc 后立刻放. 多 bot grab 不同 sid 不互锁.
    let sess_arc = {
        let map = SESSIONS.lock().unwrap();
        match map.get(&sid) {
            Some(a) => a.clone(),
            None => {
                set_err("invalid session id");
                return ERR_INVALID_SESSION;
            }
        }
    };
    let mut s = sess_arc.lock().unwrap();

    // TryGetNextFrame: WGC 在没新帧时返 Ok(default), Err 大概率是真错 (device lost /
    // session closed / access lost). 旧版全吞为 NOT_READY 会让 Go 端死等 400ms 才超时,
    // 真坏了无法区分.
    // 例外: 某些驱动/系统组合下 occluded / RDP / minimized 偶尔会返 benign HRESULT,
    // 这些归类成 ERR_NOT_READY 让 caller 自然 retry, 避免刷 GRAB_FAILED 风暴.
    let frame = match s.frame_pool.TryGetNextFrame() {
        Ok(f) => f,
        Err(e) => {
            let hr = e.code();
            if hr == DXGI_ERROR_WAIT_TIMEOUT || hr == DXGI_STATUS_OCCLUDED {
                return ERR_NOT_READY;
            }
            set_err(format!(
                "TryGetNextFrame: {:#x} {}",
                hr.0 as u32,
                e.message()
            ));
            return ERR_GRAB_FAILED;
        }
    };

    let content_size = match frame.ContentSize() {
        Ok(sz) => sz,
        Err(e) => {
            let _ = frame.Close();
            set_err(format!("content size: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };
    let w = content_size.Width;
    let h = content_size.Height;
    // 最小化/未渲染窗口 ContentSize 可能 0×0. CreateTexture2D 拒收 0 宽高 (旧版会刷
    // CreateTexture2D 失败日志), 直接当 "没新帧" 让 caller retry.
    if w <= 0 || h <= 0 {
        let _ = frame.Close();
        return ERR_NOT_READY;
    }
    let needed = (w as u32) * (h as u32) * 4;

    unsafe {
        if !out_w.is_null() {
            *out_w = w;
        }
        if !out_h.is_null() {
            *out_h = h;
        }
        if buf.is_null() || buf_len < needed {
            let _ = frame.Close();
            return ERR_BUF_TOO_SMALL;
        }
    }

    // Frame surface (WinRT IDirect3DSurface) → ID3D11Texture2D.
    // WinRT 类型不能直接 QI 到 IDXGISurface / ID3D11Texture2D, 要走 interop interface
    // IDirect3DDxgiInterfaceAccess::GetInterface<T>() 才能拿到 D3D11 资源.
    let surface = match frame.Surface() {
        Ok(s) => s,
        Err(e) => {
            let _ = frame.Close();
            set_err(format!("surface: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };
    let access: IDirect3DDxgiInterfaceAccess = match surface.cast() {
        Ok(a) => a,
        Err(e) => {
            let _ = frame.Close();
            set_err(format!("dxgi interop access: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };
    let source_tex: ID3D11Texture2D = match unsafe { access.GetInterface::<ID3D11Texture2D>() } {
        Ok(t) => t,
        Err(e) => {
            let _ = frame.Close();
            set_err(format!("get d3d11 texture: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };

    // staging texture 复用: 仅当尺寸跟当前帧不一致时重建.
    unsafe {
        // reborrow 出 &mut Session 让 borrow checker 看见 disjoint fields
        // (MutexGuard 自身不直接支持 split borrow).
        let inner: &mut Session = &mut s;
        if inner.staging.is_none() || inner.staging_w != w || inner.staging_h != h {
            let mut desc = D3D11_TEXTURE2D_DESC::default();
            source_tex.GetDesc(&mut desc);
            desc.Usage = D3D11_USAGE_STAGING;
            desc.CPUAccessFlags = D3D11_CPU_ACCESS_READ.0 as u32;
            desc.BindFlags = 0;
            desc.MiscFlags = 0;

            let mut new_staging: Option<ID3D11Texture2D> = None;
            if let Err(e) = inner
                .d3d_device
                .CreateTexture2D(&desc, None, Some(&mut new_staging))
            {
                let _ = frame.Close();
                set_err(format!("CreateTexture2D: {}", e.message()));
                return ERR_GRAB_FAILED;
            }
            match new_staging {
                Some(t) => {
                    inner.staging = Some(t);
                    inner.staging_w = w;
                    inner.staging_h = h;
                }
                None => {
                    let _ = frame.Close();
                    return ERR_GRAB_FAILED;
                }
            }
        }
        // split borrow: staging 跟 d3d_context 是 Session 的 disjoint fields,
        // 都不可变借用, Rust NLL 接受.
        let staging_ref = inner.staging.as_ref().unwrap();
        inner.d3d_context.CopyResource(staging_ref, &source_tex);

        // Map 读出 BGRA. D3D11 Map 行有 RowPitch padding (>=W*4), 要逐行拷贝.
        let mut mapped = D3D11_MAPPED_SUBRESOURCE::default();
        if let Err(e) = inner
            .d3d_context
            .Map(staging_ref, 0, D3D11_MAP_READ, 0, Some(&mut mapped))
        {
            let _ = frame.Close();
            set_err(format!("Map: {}", e.message()));
            return ERR_MAP_FAILED;
        }
        let row_bytes = (w as usize) * 4;
        let src = mapped.pData as *const u8;
        let pitch = mapped.RowPitch as usize;
        for y in 0..(h as usize) {
            let src_row = src.add(y * pitch);
            let dst_row = buf.add(y * row_bytes);
            std::ptr::copy_nonoverlapping(src_row, dst_row, row_bytes);
        }
        inner.d3d_context.Unmap(staging_ref, 0);
    }

    // 显式 drop 中间 COM ref, 确保 FramePool buffer 被释放前没人引用 (某些 WGC 驱动
    // 直到最后一个 ref release 才真还 buffer).
    drop(source_tex);
    drop(access);
    drop(surface);
    // 显式 Close frame 还回 FramePool buffer (count=2, 不还满了下次抓不到).
    // 必须在 Recreate 之前 — 老 frame 还引用 pool 时 Recreate 会 stuck / device removed
    // (WGC 经典坑).
    let _ = frame.Close();

    // 当前帧尺寸跟 pool 配置不一致 (窗口 resize 过) → recreate 让下次抓更准.
    // 任一步骤失败保留旧 pool 尺寸下次再试, 不 panic (旧版三个 unwrap 会真崩进程).
    if w != s.pool_w || h != s.pool_h {
        let new_size = SizeInt32 {
            Width: w,
            Height: h,
        };
        let recreate_result: windows::core::Result<()> = (|| unsafe {
            let dxgi: IDXGIDevice = s.d3d_device.cast()?;
            let inspectable: IInspectable = CreateDirect3D11DeviceFromDXGIDevice(&dxgi)?;
            let direct3d: windows::Graphics::DirectX::Direct3D11::IDirect3DDevice =
                inspectable.cast()?;
            s.frame_pool.Recreate(
                &direct3d,
                DirectXPixelFormat::B8G8R8A8UIntNormalized,
                2,
                new_size,
            )
        })();
        if recreate_result.is_ok() {
            s.pool_w = w;
            s.pool_h = h;
        }
    }

    OK
}

/// 释放会话.
#[no_mangle]
pub extern "C" fn wgc_session_close(sid: u64) {
    let arc = SESSIONS.lock().unwrap().remove(&sid);
    if let Some(a) = arc {
        // poison 也照样 Close, 资源不能漏
        let s = a.lock().unwrap_or_else(|e| e.into_inner());
        // 先关 frame_pool 再关 session (WGC sample 推荐: pool 可能仍引用 session).
        let _ = s.frame_pool.Close();
        let _ = s.session.Close();
        // d3d_device / context / staging drop 自动释放
    }
}
