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
use windows::Win32::Foundation::{DXGI_STATUS_OCCLUDED, HWND, POINT, RECT};
use windows::Win32::Graphics::Direct3D::D3D_DRIVER_TYPE_HARDWARE;
use windows::Win32::Graphics::Dwm::{DwmGetWindowAttribute, DWMWA_EXTENDED_FRAME_BOUNDS};
use windows::Win32::Graphics::Gdi::ClientToScreen;
use windows::Win32::UI::WindowsAndMessaging::GetClientRect;
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

// COM MTA init 是 **thread-scoped** 不是 process-scoped — RoInitialize 文档明说
// "initializes Windows Runtime on the calling thread". Go runtime syscall 在不同
// OS thread 间切换 (wgc_session_open 可能在 thread A, grab 可能在 thread B), 每个
// 线程必须自己 init 一次, 否则 capture pipeline silently 不出帧 (TryGetNextFrame
// 永远返 Err(HRESULT(0))).
//
// thread_local Cell<bool> 标记本线程是否 init 过, 入口处 ensure 一次. 重复 init
// 返 S_FALSE, 不同 apartment 冲突返 RPC_E_CHANGED_MODE — 都吞掉. free-threaded
// FramePool 在两种情况下都能跑.
thread_local! {
    static THREAD_COM_INIT: std::cell::Cell<bool> = const { std::cell::Cell::new(false) };
}

fn ensure_thread_com_init() {
    THREAD_COM_INIT.with(|c| {
        if !c.get() {
            let _ = unsafe { RoInitialize(RO_INIT_MULTITHREADED) };
            c.set(true);
        }
    });
}

struct Session {
    // 保留 HWND 给 grab 算客户区裁剪用. CreateForWindow 抓的是整个 window content
    // (含标题栏 + 边框), 上层 ROI 按客户区算, 直接喂就错位 — grab 内拷贝时按 client
    // offset 跳过非客户区, 上层零感知.
    // 存 isize 而非 HWND: HWND 裸指针不 Sync, Arc<Mutex<Session>> 要 Sync. 用时转 HWND.
    hwnd_raw: isize,
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
}

// ClientOffset 描述客户区在 WGC frame 内的位置.
// off_x/off_y = 客户区左上角相对帧左上角的偏移 (px).
// client_w/client_h = 客户区尺寸 (要拷给上层的).
struct ClientOffset {
    off_x: i32,
    off_y: i32,
    client_w: i32,
    client_h: i32,
}

// compute_client_offset 算客户区在 WGC frame 内的偏移 + 尺寸.
//
// 关键: WGC 抓的不是 GetWindowRect (那个 Win10+ 含 invisible DWM shadow), 而是
// DwmGetWindowAttribute(DWMWA_EXTENDED_FRAME_BOUNDS) 给的"真实可见"矩形.
// 实测异环 1920x1080: GetWindowRect=1936x1119 (含 8/8/7 shadow), WGC frame=1922x1112,
// DWM extended bounds = 1922x1112 跟 WGC 一致. 用 GetWindowRect 算 offset 会越界.
//
// 失败 (窗口无效 / DWM 不可用 / size 异常) 返 None, 调用方 fallback 用全 frame 不裁.
fn compute_client_offset(hwnd: HWND) -> Option<ClientOffset> {
    unsafe {
        let mut frame_rect = RECT::default();
        let res = DwmGetWindowAttribute(
            hwnd,
            DWMWA_EXTENDED_FRAME_BOUNDS,
            &mut frame_rect as *mut RECT as *mut c_void,
            std::mem::size_of::<RECT>() as u32,
        );
        if res.is_err() {
            return None;
        }
        let mut cr = RECT::default();
        if GetClientRect(hwnd, &mut cr).is_err() {
            return None;
        }
        let mut origin = POINT { x: 0, y: 0 };
        if !ClientToScreen(hwnd, &mut origin).as_bool() {
            return None;
        }
        let client_w = cr.right - cr.left;
        let client_h = cr.bottom - cr.top;
        if client_w <= 0 || client_h <= 0 {
            return None;
        }
        Some(ClientOffset {
            off_x: origin.x - frame_rect.left,
            off_y: origin.y - frame_rect.top,
            client_w,
            client_h,
        })
    }
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
        // buffer count=1: 微软文档 "If you only need the latest frame, use 1". count=2
        // 时 pool 满后 capture 丢新帧, TryGetNextFrame 拿最老的 — 我们间隔抓帧 (500ms+)
        // 会让 pool 长期满, 永远拿 stale frame (bug: cache session 重测 5 帧 pix 完全
        // 一样, nocache 每次新 session 反而能拿新帧 — 因为新 session 启动后第一帧总是
        // 新鲜的).
        let size = item.Size()?;
        let frame_pool = Direct3D11CaptureFramePool::CreateFreeThreaded(
            &direct3d_device,
            DirectXPixelFormat::B8G8R8A8UIntNormalized,
            1,
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
            hwnd_raw: hwnd.0 as isize,
            d3d_device,
            d3d_context,
            item,
            frame_pool,
            session,
            pool_w: size.Width,
            pool_h: size.Height,
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
    ensure_thread_com_init();
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
    // Go runtime 可能在不同 OS thread 调 grab, 每个 thread 独立 init COM.
    ensure_thread_com_init();
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

    // Drain pool 拿最新 frame.
    //
    // 单次 TryGetNextFrame 在 cache session 复用模式下实测会返"老帧" — 我们间隔
    // 抓帧 (500ms+) 比 capture pipeline 推帧 (vsync ~16ms) 慢得多, pool 长期满
    // (即使 count=1), 第一次 TryGetNextFrame 拿到的是 pool 里最老那一帧.
    //
    // 循环 TryGetNextFrame 直到 Err, 拿最后一个 (=最新). 微软文档说
    // "TryGetNextFrame after a frame has been received will release the frame back
    // to the pool", 所以中间的 frame 不需要显式 Close (drop 即可, TryGetNextFrame
    // 内部已 release 上一个回 pool).
    //
    // benign HRESULT 处理: 第一次就 Err 才报 NOT_READY/GRAB_FAILED; 已经拿到至少
    // 一帧后再 Err 是正常 drain 结束.
    let mut latest_frame: Option<windows::Graphics::Capture::Direct3D11CaptureFrame> = None;
    loop {
        match s.frame_pool.TryGetNextFrame() {
            Ok(f) => {
                latest_frame = Some(f);
            }
            Err(e) => {
                if latest_frame.is_some() {
                    break; // drain 结束, 用 latest
                }
                let hr = e.code();
                if hr.0 == 0 || hr == DXGI_ERROR_WAIT_TIMEOUT || hr == DXGI_STATUS_OCCLUDED {
                    return ERR_NOT_READY;
                }
                set_err(format!(
                    "TryGetNextFrame: {:#x} {}",
                    hr.0 as u32,
                    e.message()
                ));
                return ERR_GRAB_FAILED;
            }
        }
    }
    let frame = latest_frame.unwrap();

    let content_size = match frame.ContentSize() {
        Ok(sz) => sz,
        Err(e) => {
            let _ = frame.Close();
            set_err(format!("content size: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };
    let frame_w = content_size.Width;
    let frame_h = content_size.Height;
    // 最小化/未渲染窗口 ContentSize 可能 0×0. CreateTexture2D 拒收 0 宽高 (旧版会刷
    // CreateTexture2D 失败日志), 直接当 "没新帧" 让 caller retry.
    if frame_w <= 0 || frame_h <= 0 {
        let _ = frame.Close();
        return ERR_NOT_READY;
    }

    // CreateForWindow 抓的是整个 window content (含标题栏 + 边框), 不是客户区.
    // 算客户区在 frame 内的偏移, 拷贝时只拷客户区, 上层 ROI 按客户区算就不错位.
    // 算失败或越界 fallback 用全 frame (不 crop), 让上层至少能跑.
    let (out_pix_w, out_pix_h, src_off_x, src_off_y) =
        match compute_client_offset(HWND(s.hwnd_raw as *mut c_void)) {
            Some(co)
                if co.off_x >= 0
                    && co.off_y >= 0
                    && co.off_x + co.client_w <= frame_w
                    && co.off_y + co.client_h <= frame_h =>
            {
                (co.client_w, co.client_h, co.off_x, co.off_y)
            }
            _ => (frame_w, frame_h, 0, 0),
        };
    let needed = (out_pix_w as u32) * (out_pix_h as u32) * 4;

    unsafe {
        if !out_w.is_null() {
            *out_w = out_pix_w;
        }
        if !out_h.is_null() {
            *out_h = out_pix_h;
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

    // staging texture 每帧重建 (不缓存复用).
    //
    // 历史教训: 第一版 staging 跨 grab 复用同一 texture, 实测 cache session 模式下连续
    // grab 都拿到第一次 grab 的内容 (5 round pix 完全一样, fishing 状态机看 stale 帧
    // 走错分支). 推测 WGC frame pool + D3D11 staging 复用在 free-threaded + 间隔 grab
    // 模式下 driver 行为异常 — 没 dig 到 root cause, 但每帧重建实测稳定. 1-2ms/frame
    // overhead 可接受.
    unsafe {
        let mut desc = D3D11_TEXTURE2D_DESC::default();
        source_tex.GetDesc(&mut desc);
        desc.Usage = D3D11_USAGE_STAGING;
        desc.CPUAccessFlags = D3D11_CPU_ACCESS_READ.0 as u32;
        desc.BindFlags = 0;
        desc.MiscFlags = 0;

        let mut staging_opt: Option<ID3D11Texture2D> = None;
        if let Err(e) = s.d3d_device.CreateTexture2D(&desc, None, Some(&mut staging_opt)) {
            let _ = frame.Close();
            set_err(format!("CreateTexture2D: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
        let staging = match staging_opt {
            Some(t) => t,
            None => {
                let _ = frame.Close();
                return ERR_GRAB_FAILED;
            }
        };
        s.d3d_context.CopyResource(&staging, &source_tex);

        // Map 读出 BGRA. D3D11 Map 行有 RowPitch padding (>=W*4), 要逐行拷贝.
        let mut mapped = D3D11_MAPPED_SUBRESOURCE::default();
        if let Err(e) = s.d3d_context.Map(&staging, 0, D3D11_MAP_READ, 0, Some(&mut mapped)) {
            let _ = frame.Close();
            set_err(format!("Map: {}", e.message()));
            return ERR_MAP_FAILED;
        }
        // 拷客户区: src 起点偏 src_off_y 行 + src_off_x*4 字节, 行宽 out_pix_w*4,
        // 共 out_pix_h 行. dst 连续无 padding. fallback 模式 src_off=0 + out_pix=frame_size
        // 等价于全 frame 拷贝.
        let row_bytes = (out_pix_w as usize) * 4;
        let src = mapped.pData as *const u8;
        let pitch = mapped.RowPitch as usize;
        let src_origin = src.add(src_off_y as usize * pitch + src_off_x as usize * 4);
        for y in 0..(out_pix_h as usize) {
            let src_row = src_origin.add(y * pitch);
            let dst_row = buf.add(y * row_bytes);
            std::ptr::copy_nonoverlapping(src_row, dst_row, row_bytes);
        }
        s.d3d_context.Unmap(&staging, 0);
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
    // 注意: pool 配置用全 frame 尺寸 (frame_w/h) 而不是裁后客户区, 跟 source_tex 一致.
    // 任一步骤失败保留旧 pool 尺寸下次再试, 不 panic (旧版三个 unwrap 会真崩进程).
    if frame_w != s.pool_w || frame_h != s.pool_h {
        let new_size = SizeInt32 {
            Width: frame_w,
            Height: frame_h,
        };
        let recreate_result: windows::core::Result<()> = (|| unsafe {
            let dxgi: IDXGIDevice = s.d3d_device.cast()?;
            let inspectable: IInspectable = CreateDirect3D11DeviceFromDXGIDevice(&dxgi)?;
            let direct3d: windows::Graphics::DirectX::Direct3D11::IDirect3DDevice =
                inspectable.cast()?;
            s.frame_pool.Recreate(
                &direct3d,
                DirectXPixelFormat::B8G8R8A8UIntNormalized,
                1, // 跟 open 一致, 只 hold 最新帧
                new_size,
            )
        })();
        if recreate_result.is_ok() {
            s.pool_w = frame_w;
            s.pool_h = frame_h;
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
