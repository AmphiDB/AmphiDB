# MySQL Manager 诊断脚本 (PowerShell)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "MySQL Manager 诊断工具" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. 检查 Windows 版本
Write-Host "[1/7] 检查 Windows 版本..." -ForegroundColor Yellow
$osInfo = Get-CimInstance Win32_OperatingSystem
Write-Host "  操作系统: $($osInfo.Caption)" -ForegroundColor Gray
Write-Host "  版本: $($osInfo.Version)" -ForegroundColor Gray
Write-Host "  架构: $($osInfo.OSArchitecture)" -ForegroundColor Gray
Write-Host ""

# 2. 检查 WebView2
Write-Host "[2/7] 检查 WebView2 运行时..." -ForegroundColor Yellow
$webview2Path = "HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
try {
    $webview2 = Get-ItemProperty -Path $webview2Path -Name pv -ErrorAction Stop
    Write-Host "  ✓ WebView2 已安装" -ForegroundColor Green
    Write-Host "  版本: $($webview2.pv)" -ForegroundColor Gray
} catch {
    Write-Host "  ✗ WebView2 未安装" -ForegroundColor Red
    Write-Host "  请下载并安装: https://developer.microsoft.com/microsoft-edge/webview2/" -ForegroundColor Yellow
}
Write-Host ""

# 3. 检查 Visual C++ Redistributable
Write-Host "[3/7] 检查 Visual C++ Redistributable..." -ForegroundColor Yellow
$vcRedistPath = "HKLM:\SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64"
try {
    $vcRedist = Get-ItemProperty -Path $vcRedistPath -ErrorAction Stop
    Write-Host "  ✓ Visual C++ Redistributable 已安装" -ForegroundColor Green
    Write-Host "  版本: $($vcRedist.Version)" -ForegroundColor Gray
} catch {
    Write-Host "  ✗ Visual C++ Redistributable 可能未安装" -ForegroundColor Red
    Write-Host "  请下载并安装: https://aka.ms/vs/17/release/vc_redist.x64.exe" -ForegroundColor Yellow
}
Write-Host ""

# 4. 检查可执行文件
Write-Host "[4/7] 检查可执行文件..." -ForegroundColor Yellow
$exePath = "MySQL-Manager.exe"
if (Test-Path $exePath) {
    $fileInfo = Get-Item $exePath
    Write-Host "  ✓ MySQL-Manager.exe 存在" -ForegroundColor Green
    Write-Host "  大小: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor Gray
    Write-Host "  修改时间: $($fileInfo.LastWriteTime)" -ForegroundColor Gray
    
    # 计算哈希
    Write-Host "  计算 SHA256 哈希..." -ForegroundColor Gray
    $hash = Get-FileHash $exePath -Algorithm SHA256
    Write-Host "  SHA256: $($hash.Hash)" -ForegroundColor Gray
} else {
    Write-Host "  ✗ MySQL-Manager.exe 不存在" -ForegroundColor Red
    Write-Host "  当前目录: $(Get-Location)" -ForegroundColor Yellow
}
Write-Host ""

# 5. 检查防火墙
Write-Host "[5/7] 检查 Windows 防火墙..." -ForegroundColor Yellow
try {
    $firewallProfiles = Get-NetFirewallProfile
    foreach ($profile in $firewallProfiles) {
        Write-Host "  $($profile.Name): $($profile.Enabled)" -ForegroundColor Gray
    }
} catch {
    Write-Host "  无法检查防火墙状态" -ForegroundColor Yellow
}
Write-Host ""

# 6. 检查 Windows Defender
Write-Host "[6/7] 检查 Windows Defender..." -ForegroundColor Yellow
try {
    $defender = Get-MpComputerStatus
    Write-Host "  实时保护: $($defender.RealTimeProtectionEnabled)" -ForegroundColor Gray
    Write-Host "  反恶意软件: $($defender.AntivirusEnabled)" -ForegroundColor Gray
} catch {
    Write-Host "  无法检查 Windows Defender 状态" -ForegroundColor Yellow
}
Write-Host ""

# 7. 检查进程
Write-Host "[7/7] 检查是否有运行中的实例..." -ForegroundColor Yellow
$processes = Get-Process | Where-Object { $_.ProcessName -like "*MySQL-Manager*" }
if ($processes) {
    Write-Host "  ✓ 发现运行中的实例:" -ForegroundColor Green
    foreach ($proc in $processes) {
        Write-Host "    PID: $($proc.Id), 内存: $([math]::Round($proc.WorkingSet64 / 1MB, 2)) MB" -ForegroundColor Gray
    }
} else {
    Write-Host "  没有运行中的实例" -ForegroundColor Gray
}
Write-Host ""

# 尝试启动应用
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "尝试启动应用..." -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if (Test-Path $exePath) {
    Write-Host "正在启动 MySQL-Manager.exe..." -ForegroundColor Yellow
    Write-Host "如果应用没有启动，请查看错误信息" -ForegroundColor Yellow
    Write-Host ""
    
    try {
        $process = Start-Process -FilePath $exePath -PassThru -ErrorAction Stop
        Start-Sleep -Seconds 2
        
        if ($process.HasExited) {
            Write-Host "✗ 应用启动后立即退出" -ForegroundColor Red
            Write-Host "  退出代码: $($process.ExitCode)" -ForegroundColor Red
        } else {
            Write-Host "✓ 应用似乎已成功启动" -ForegroundColor Green
            Write-Host "  进程 ID: $($process.Id)" -ForegroundColor Gray
        }
    } catch {
        Write-Host "✗ 启动失败: $($_.Exception.Message)" -ForegroundColor Red
    }
} else {
    Write-Host "✗ 无法找到 MySQL-Manager.exe" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "诊断完成" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "如果应用仍然无法启动，请:" -ForegroundColor Yellow
Write-Host "1. 安装 WebView2 (如果未安装)" -ForegroundColor White
Write-Host "2. 安装 Visual C++ Redistributable (如果未安装)" -ForegroundColor White
Write-Host "3. 检查防火墙/杀毒软件设置" -ForegroundColor White
Write-Host "4. 以管理员身份运行此脚本" -ForegroundColor White
Write-Host "5. 查看 WINDOWS_EXE_TROUBLESHOOTING.md 获取详细帮助" -ForegroundColor White
Write-Host ""

# 检查事件日志
Write-Host "检查最近的应用程序错误..." -ForegroundColor Yellow
try {
    $errors = Get-EventLog -LogName Application -EntryType Error -Newest 5 -ErrorAction SilentlyContinue | 
              Where-Object { $_.Source -like "*MySQL*" -or $_.Message -like "*MySQL-Manager*" }
    if ($errors) {
        Write-Host "发现相关错误:" -ForegroundColor Red
        foreach ($error in $errors) {
            Write-Host "  时间: $($error.TimeGenerated)" -ForegroundColor Gray
            Write-Host "  来源: $($error.Source)" -ForegroundColor Gray
            Write-Host "  消息: $($error.Message.Substring(0, [Math]::Min(200, $error.Message.Length)))..." -ForegroundColor Gray
            Write-Host ""
        }
    }
} catch {
    # 忽略错误
}

Write-Host "按任意键退出..." -ForegroundColor Gray
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
