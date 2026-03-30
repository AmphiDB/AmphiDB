@echo off
chcp 65001 >nul
echo ========================================
echo MySQL Manager 诊断工具
echo ========================================
echo.

echo [1/6] 检查 Windows 版本...
ver
echo.

echo [2/6] 检查 WebView2 运行时...
reg query "HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv 2>nul
if %ERRORLEVEL% EQU 0 (
    echo ✓ WebView2 已安装
) else (
    echo ✗ WebView2 未安装
    echo.
    echo 请下载并安装 WebView2:
    echo https://developer.microsoft.com/microsoft-edge/webview2/
)
echo.

echo [3/6] 检查 Visual C++ Redistributable...
reg query "HKLM\SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64" /v Version 2>nul
if %ERRORLEVEL% EQU 0 (
    echo ✓ Visual C++ Redistributable 已安装
) else (
    echo ✗ Visual C++ Redistributable 可能未安装
    echo.
    echo 请下载并安装:
    echo https://aka.ms/vs/17/release/vc_redist.x64.exe
)
echo.

echo [4/6] 检查可执行文件...
if exist "MySQL-Manager.exe" (
    echo ✓ MySQL-Manager.exe 存在
    dir MySQL-Manager.exe | find "MySQL-Manager.exe"
) else (
    echo ✗ MySQL-Manager.exe 不存在
    echo 请确保在正确的目录运行此脚本
)
echo.

echo [5/6] 检查防火墙状态...
netsh advfirewall show currentprofile state | find "State"
echo.

echo [6/6] 尝试运行应用...
echo 正在启动 MySQL-Manager.exe...
echo 如果应用没有启动，请查看下方的错误代码
echo.

if exist "MySQL-Manager.exe" (
    start "" "MySQL-Manager.exe"
    timeout /t 3 >nul
    echo 错误代码: %ERRORLEVEL%
    if %ERRORLEVEL% NEQ 0 (
        echo.
        echo 应用启动失败，错误代码: %ERRORLEVEL%
    )
) else (
    echo 无法找到 MySQL-Manager.exe
)

echo.
echo ========================================
echo 诊断完成
echo ========================================
echo.
echo 如果应用仍然无法启动，请:
echo 1. 安装 WebView2 (如果未安装)
echo 2. 安装 Visual C++ Redistributable (如果未安装)
echo 3. 检查防火墙/杀毒软件设置
echo 4. 以管理员身份运行此脚本
echo 5. 查看 WINDOWS_EXE_TROUBLESHOOTING.md 获取详细帮助
echo.

pause
