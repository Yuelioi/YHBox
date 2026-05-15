// capture_wgc: 用 Windows Graphics Capture (WGC) + D3D11 给单个 HWND 抓帧。
//
// 暴露给 Go 端 4 个 C ABI 函数（详见 ../README 或 pkg/capture/wgc_windows.go）。
// 一个 session 对应一个 HWND；多 bot 各开各的 session 互不干扰，DLL 内部用
// HashMap<u64, Session> + Mutex 保护。
//
// 帧抓取走 FramePool::try_get_next_frame() 同步轮询：caller 调 grab 时如果还
// 没新帧就返回 ERR_NOT_READY，让 Go 那边按自己的节奏重试。这样 DLL 这边不需要
// 给 WinRT 注册 callback（Go syscall 接 callback 会麻烦很多）。

use once_cell::sync::Lazy;
use std::cell::RefCell;
use std::collections::HashMap;
use std::ffi::{c_char, c_void, CStr, CString};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Mutex;

use windows::core::{IInspectable, Interface};
use windows::Graphics::Capture::{Direct3D11CaptureFramePool, GraphicsCaptureItem};
use windows::Graphics::DirectX::DirectXPixelFormat;
use windows::Graphics::SizeInt32;
use windows::Win32::Foundation::HWND;
use windows::Win32::Graphics::Direct3D::D3D_DRIVER_TYPE_HARDWARE;
use windows::Win32::Graphics::Direct3D11::{
    D3D11CreateDevice, ID3D11Device, ID3D11DeviceContext, ID3D11Texture2D, D3D11_CPU_ACCESS_READ,
    D3D11_CREATE_DEVICE_BGRA_SUPPORT, D3D11_MAPPED_SUBRESOURCE, D3D11_MAP_READ, D3D11_SDK_VERSION,
    D3D11_TEXTURE2D_DESC, D3D11_USAGE_STAGING,
};
use windows::Wdk::System::SystemServices::RtlGetVersion;
use windows::Win32::Graphics::Dxgi::IDXGIDevice;
use windows::Win32::System::SystemInformation::OSVERSIONINFOW;
use windows::Win32::System::WinRT::Direct3D11::{
    CreateDirect3D11DeviceFromDXGIDevice, IDirect3DDxgiInterfaceAccess,
};
use windows::Win32::System::WinRT::Graphics::Capture::IGraphicsCaptureItemInterop;

// 返回码约定（i32）
pub const OK: i32 = 0;
pub const ERR_BUF_TOO_SMALL: i32 = -1;
pub const ERR_NOT_READY: i32 = -2;
pub const ERR_INVALID_SESSION: i32 = -11;
pub const ERR_GRAB_FAILED: i32 = -12;
pub const ERR_MAP_FAILED: i32 = -13;

thread_local! {
    // 每个线程一份 last error 字符串，避免跨线程 race。Go 端拿到 sid==0
    // 或负 return 后立刻读，跟调用是同线程语义。
    static LAST_ERR: RefCell<CString> = RefCell::new(CString::new("").unwrap());
}

fn set_err<S: AsRef<str>>(msg: S) {
    let s = CString::new(msg.as_ref()).unwrap_or_else(|_| CString::new("err").unwrap());
    LAST_ERR.with(|c| *c.borrow_mut() = s);
}

struct Session {
    d3d_device: ID3D11Device,
    d3d_context: ID3D11DeviceContext,
    // FramePool 内部不持有 GraphicsCaptureItem 的强引用——必须自己 keep alive，
    // 否则 capture session 会突然停止。这里挂着不动就行。
    #[allow(dead_code)]
    item: GraphicsCaptureItem,
    frame_pool: Direct3D11CaptureFramePool,
    session: windows::Graphics::Capture::GraphicsCaptureSession,
    // 当前帧池配置的尺寸；窗口被 resize 时需要 recreate frame pool
    pool_w: i32,
    pool_h: i32,
}

// 全局 session 表。AtomicU64 单调递增分配 sid。
static SESSIONS: Lazy<Mutex<HashMap<u64, Session>>> = Lazy::new(|| Mutex::new(HashMap::new()));
static NEXT_SID: AtomicU64 = AtomicU64::new(1);

// 创建 D3D11 + WinRT IDirect3DDevice，并构造 GraphicsCaptureItem + FramePool。
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

        // SetIsCursorCaptureEnabled 从 Win10 1903 (build 18362) 就支持，直接调.
        let _ = session.SetIsCursorCaptureEnabled(false);

        // SetIsBorderRequired 真正生效要 Win11 / Server 2022 (build 20348+).
        // Win10 任何版本调用都被系统忽略，黄框无法关闭 (微软硬性 UI 提示).
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
        })
    }
}

/// 返回最近一次错误描述（C 字符串，UTF-8）。线程本地，每线程一份。
/// 调用方不要 free 这个指针。
#[no_mangle]
pub extern "C" fn wgc_last_error() -> *const c_char {
    LAST_ERR.with(|c| c.borrow().as_ptr())
}

/// 为 HWND 开抓帧会话。
/// 成功：返回非 0 sid，把客户区 W/H 写入 out_w/out_h。
/// 失败：返回 0，错误细节看 wgc_last_error()。
#[no_mangle]
pub extern "C" fn wgc_session_open(
    hwnd: u64,
    out_w: *mut i32,
    out_h: *mut i32,
) -> u64 {
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
            SESSIONS.lock().unwrap().insert(sid, s);
            sid
        }
        Err(e) => {
            set_err(format!("open: {}", e.message()));
            0
        }
    }
}

/// 抓最新帧到 buf（BGRA，行无 padding）。
/// 返回 OK / ERR_BUF_TOO_SMALL / ERR_NOT_READY / ERR_*。
/// out_w/out_h 在 OK 和 ERR_BUF_TOO_SMALL 时都写入当前帧尺寸。
#[no_mangle]
pub extern "C" fn wgc_session_grab(
    sid: u64,
    buf: *mut u8,
    buf_len: u32,
    out_w: *mut i32,
    out_h: *mut i32,
) -> i32 {
    let mut sessions = SESSIONS.lock().unwrap();
    let Some(s) = sessions.get_mut(&sid) else {
        set_err("invalid session id");
        return ERR_INVALID_SESSION;
    };

    // try_get_next_frame: 没新帧返回 None / 默认值
    let frame = match s.frame_pool.TryGetNextFrame() {
        Ok(f) => f,
        Err(_) => return ERR_NOT_READY,
    };

    // 帧尺寸（可能因窗口 resize 跟 pool 不一致；先以 frame 自己的 size 为准）
    let content_size = match frame.ContentSize() {
        Ok(sz) => sz,
        Err(e) => {
            set_err(format!("content size: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };
    let w = content_size.Width;
    let h = content_size.Height;
    let needed = (w as u32) * (h as u32) * 4;

    unsafe {
        if !out_w.is_null() {
            *out_w = w;
        }
        if !out_h.is_null() {
            *out_h = h;
        }
        if buf.is_null() || buf_len < needed {
            return ERR_BUF_TOO_SMALL;
        }
    }

    // Frame surface (WinRT IDirect3DSurface) → ID3D11Texture2D。
    // WinRT 类型不能直接 QI 到 IDXGISurface / ID3D11Texture2D，要走 interop interface
    // IDirect3DDxgiInterfaceAccess::GetInterface<T>() 才能拿到 D3D11 资源.
    let surface = match frame.Surface() {
        Ok(s) => s,
        Err(e) => {
            set_err(format!("surface: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };
    let access: IDirect3DDxgiInterfaceAccess = match surface.cast() {
        Ok(a) => a,
        Err(e) => {
            set_err(format!("dxgi interop access: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };
    let source_tex: ID3D11Texture2D = match unsafe { access.GetInterface::<ID3D11Texture2D>() } {
        Ok(t) => t,
        Err(e) => {
            set_err(format!("get d3d11 texture: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
    };

    // 创建 staging texture（CPU 可读），CopyResource 从 source 拷贝过来
    unsafe {
        let mut desc = D3D11_TEXTURE2D_DESC::default();
        source_tex.GetDesc(&mut desc);
        desc.Usage = D3D11_USAGE_STAGING;
        desc.CPUAccessFlags = D3D11_CPU_ACCESS_READ.0 as u32;
        desc.BindFlags = 0;
        desc.MiscFlags = 0;

        let mut staging: Option<ID3D11Texture2D> = None;
        if let Err(e) = s
            .d3d_device
            .CreateTexture2D(&desc, None, Some(&mut staging))
        {
            set_err(format!("CreateTexture2D: {}", e.message()));
            return ERR_GRAB_FAILED;
        }
        let staging = match staging {
            Some(t) => t,
            None => return ERR_GRAB_FAILED,
        };
        s.d3d_context.CopyResource(&staging, &source_tex);

        // Map 读出 BGRA
        let mut mapped = D3D11_MAPPED_SUBRESOURCE::default();
        if let Err(e) = s
            .d3d_context
            .Map(&staging, 0, D3D11_MAP_READ, 0, Some(&mut mapped))
        {
            set_err(format!("Map: {}", e.message()));
            return ERR_MAP_FAILED;
        }
        // 注意：D3D11 Map 返回的行有 RowPitch padding (>=W*4)，要逐行拷贝
        let row_bytes = (w as usize) * 4;
        let src = mapped.pData as *const u8;
        let pitch = mapped.RowPitch as usize;
        for y in 0..(h as usize) {
            let src_row = src.add(y * pitch);
            let dst_row = buf.add(y * row_bytes);
            std::ptr::copy_nonoverlapping(src_row, dst_row, row_bytes);
        }
        s.d3d_context.Unmap(&staging, 0);
    }

    // 如果当前帧尺寸跟 pool 配置不一致（窗口 resize 过），recreate 一下让下次抓更准
    if w != s.pool_w || h != s.pool_h {
        let new_size = SizeInt32 {
            Width: w,
            Height: h,
        };
        if let Ok(()) = unsafe {
            let dxgi: IDXGIDevice = s.d3d_device.cast().unwrap();
            let inspectable: IInspectable =
                CreateDirect3D11DeviceFromDXGIDevice(&dxgi).unwrap();
            let direct3d: windows::Graphics::DirectX::Direct3D11::IDirect3DDevice =
                inspectable.cast().unwrap();
            s.frame_pool.Recreate(
                &direct3d,
                DirectXPixelFormat::B8G8R8A8UIntNormalized,
                2,
                new_size,
            )
        } {
            s.pool_w = w;
            s.pool_h = h;
        }
    }

    OK
}

/// 释放会话。
#[no_mangle]
pub extern "C" fn wgc_session_close(sid: u64) {
    if let Some(s) = SESSIONS.lock().unwrap().remove(&sid) {
        let _ = s.session.Close();
        let _ = s.frame_pool.Close();
        // d3d_device / context drop 自动释放
        drop(s);
    }
}

// 让编译器知道 wgc_last_error 的 c_char 用法
#[allow(dead_code)]
fn _ensure_cstr() -> &'static CStr {
    CStr::from_bytes_with_nul(b"\0").unwrap()
}
