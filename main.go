package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// checkPort 检查指定端口是否被占用
func checkPort(port int) (bool, string) {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", address)
	if err == nil {
		// 能连接成功，说明端口被占用
		conn.Close()
		return true, fmt.Sprintf("端口 %d 已被占用", port)
	}
	// 连接失败，说明端口可用
	return false, fmt.Sprintf("端口 %d 可用", port)
}

// getProcessByPort 获取占用指定端口的进程信息
func getProcessByPort(port int) (pid int, processName string, err error) {
	// 使用 netstat 命令查找占用端口的进程
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return 0, "", err
	}

	lines := strings.Split(string(output), "\n")
	target := fmt.Sprintf(":%d", port)

	for _, line := range lines {
		if strings.Contains(line, target) && strings.Contains(line, "LISTENING") {
			// 提取 PID（最后一列）
			parts := strings.Fields(line)
			if len(parts) > 0 {
				pidStr := parts[len(parts)-1]
				pid, err = strconv.Atoi(pidStr)
				if err != nil {
					return 0, "", err
				}
				// 获取进程名
				processName, err = getProcessName(pid)
				if err != nil {
					return pid, fmt.Sprintf("PID: %d", pid), nil
				}
				return pid, processName, nil
			}
		}
	}
	return 0, "", fmt.Errorf("未找到占用端口 %d 的进程", port)
}

// getProcessName 获取进程名称
func getProcessName(pid int) (string, error) {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return "", fmt.Errorf("未找到进程")
	}

	// CSV 格式: "进程名.exe","PID","会话名","会话#","内存使用"
	parts := strings.Split(line, ",")
	if len(parts) > 0 {
		// 去除引号
		name := strings.Trim(parts[0], "\"")
		return name, nil
	}
	return "", fmt.Errorf("无法解析进程名")
}

// killProcess 结束指定PID的进程
func killProcess(pid int) error {
	cmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
	return cmd.Run()
}

// DeviceStatus 设备状态
type DeviceStatus int

const (
	DeviceStatusNone DeviceStatus = iota
	DeviceStatusADB
	DeviceStatusFastboot
)

// checkADBDevices 检查是否有ADB设备连接
func checkADBDevices() ([]string, error) {
	cmd := exec.Command("./tools/adb.exe", "devices")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("执行adb命令失败: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	var devices []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "List of devices") && !strings.Contains(line, "daemon") {
			// 提取设备序列号（第一个字段）
			parts := strings.Fields(line)
			if len(parts) > 0 {
				devices = append(devices, parts[0])
			}
		}
	}
	return devices, nil
}

// checkFastbootDevices 检查是否有Fastboot设备连接
func checkFastbootDevices() ([]string, error) {
	cmd := exec.Command("./tools/fastboot.exe", "devices")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("执行fastboot命令失败: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	var devices []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// 提取设备序列号（第一个字段）
			parts := strings.Fields(line)
			if len(parts) > 0 {
				devices = append(devices, parts[0])
			}
		}
	}
	return devices, nil
}

// getDeviceStatus 获取当前设备状态
func getDeviceStatus() (DeviceStatus, []string) {
	// 先检查ADB设备
	adbDevices, err := checkADBDevices()
	if err == nil && len(adbDevices) > 0 {
		return DeviceStatusADB, adbDevices
	}

	// 再检查Fastboot设备
	fastbootDevices, err := checkFastbootDevices()
	if err == nil && len(fastbootDevices) > 0 {
		return DeviceStatusFastboot, fastbootDevices
	}

	return DeviceStatusNone, nil
}

// rebootToBootloader 重启设备到Fastboot模式
func rebootToBootloader() error {
	cmd := exec.Command("./tools/adb.exe", "reboot", "bootloader")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("执行adb reboot bootloader失败: %v", err)
	}
	return nil
}

// createPortCheckWindow 创建端口检查窗口
func createPortCheckWindow(app fyne.App) fyne.Window {
	window := app.NewWindow("黎明-刷机工具箱")
	window.Resize(fyne.NewSize(800, 500))

	// 创建停止通道用于控制轮询
	stopChan := make(chan struct{})

	// 结果显示标签
	resultLabel := widget.NewLabel("点击按钮检查端口占用情况")

	// 进程信息标签
	processLabel := widget.NewLabel("")

	// 当前占用端口的PID（用于结束进程）
	var currentPid int = 0

	// 结束进程按钮
	killButton := widget.NewButton("结束占用进程", func() {
		if currentPid == 0 {
			dialog.ShowError(fmt.Errorf("没有找到要结束的进程"), window)
			return
		}

		err := killProcess(currentPid)
		if err != nil {
			dialog.ShowError(fmt.Errorf("结束进程失败: %v", err), window)
		} else {
			dialog.ShowInformation("成功", fmt.Sprintf("已成功结束进程 PID: %d", currentPid), window)
			resultLabel.SetText("端口 5037 可用")
			processLabel.SetText("")
			currentPid = 0
		}
	})
	killButton.Hide() // 默认隐藏

	// 检查端口按钮
	checkButton := widget.NewButton("检查5037端口占用", func() {
		occupied, msg := checkPort(5037)
		if occupied {
			resultLabel.SetText(msg)

			// 获取进程信息
			pid, processName, err := getProcessByPort(5037)
			if err != nil {
				processLabel.SetText(fmt.Sprintf("获取进程信息失败: %v", err))
				killButton.Hide()
			} else {
				currentPid = pid
				processLabel.SetText(fmt.Sprintf("占用进程: %s (PID: %d)", processName, pid))
				killButton.Show()
			}
		} else {
			resultLabel.SetText(msg)
			processLabel.SetText("")
			currentPid = 0
			killButton.Hide()
		}
	})
	checkButton.Importance = widget.HighImportance

	// 左侧：端口检查区域
	leftPanel := container.NewVBox(
		widget.NewLabelWithStyle("端口检查", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		checkButton,
		resultLabel,
		processLabel,
		killButton,
	)

	// 设备状态显示
	statusLabel := widget.NewLabel("未连接")
	deviceListLabel := widget.NewLabel("")

	// 更新设备状态显示（使用 fyne.Do 在主线程中执行 UI 操作）
	updateDeviceStatus := func() {
		// 获取当前状态（可以在任何线程执行）
		status, devices := getDeviceStatus()

		// 使用 fyne.Do 将 UI 更新调度到主线程
		fyne.Do(func() {
			switch status {
			case DeviceStatusADB:
				statusLabel.SetText("ADB模式")
				deviceListLabel.SetText(fmt.Sprintf("设备: %s", strings.Join(devices, ", ")))
			case DeviceStatusFastboot:
				statusLabel.SetText("Fastboot模式")
				deviceListLabel.SetText(fmt.Sprintf("设备: %s", strings.Join(devices, ", ")))
			default:
				statusLabel.SetText("未连接")
				deviceListLabel.SetText("")
			}
		})
	}

	// 设置窗口关闭时退出
	window.SetCloseIntercept(func() {
		close(stopChan)
		app.Quit()
	})

	// 启动设备状态轮询（每5秒一次）
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				updateDeviceStatus()
			case <-stopChan:
				return
			}
		}
	}()

	// 刷新设备状态按钮
	refreshStatusBtn := widget.NewButton("刷新状态", func() {
		updateDeviceStatus()
	})

	// 设备状态面板
	statusPanel := container.NewVBox(
		container.NewHBox(
			statusLabel,
			refreshStatusBtn,
		),
		deviceListLabel,
	)

	// 右侧：功能按钮区域
	// 安装驱动按钮
	installDriverBtn := widget.NewButton("安装驱动", func() {
		driverPath := "./driver/必备驱动.exe"
		cmd := exec.Command(driverPath)
		err := cmd.Start()
		if err != nil {
			dialog.ShowError(fmt.Errorf("启动驱动安装程序失败: %v", err), window)
		} else {
			dialog.ShowInformation("提示", "驱动安装程序已启动，请按照向导完成安装", window)
		}
	})

	// 进入Fastboot按钮
	fastbootBtn := widget.NewButton("进入Fastboot", func() {
		status, _ := getDeviceStatus()
		switch status {
		case DeviceStatusADB:
			err := rebootToBootloader()
			if err != nil {
				dialog.ShowError(err, window)
			} else {
				dialog.ShowInformation("提示", "正在重启设备进入Fastboot模式...", window)
				go func() {
					for i := 0; i < 10; i++ {
						updateDeviceStatus()
						cmd := exec.Command("timeout", "/t", "2", "/nobreak")
						cmd.Run()
					}
				}()
			}
		case DeviceStatusFastboot:
			dialog.ShowInformation("提示", "设备已处于Fastboot模式", window)
		default:
			dialog.ShowError(fmt.Errorf("未检测到ADB设备，请确保设备已连接并开启USB调试"), window)
		}
	})

	fastbootdBtn := widget.NewButton("进入Fastbootd", func() {
		dialog.ShowInformation("提示", "进入Fastbootd功能开发中...", window)
	})
	bootSystemBtn := widget.NewButton("进入系统", func() {
		dialog.ShowInformation("提示", "进入系统功能开发中...", window)
	})
	switchSlotBtn := widget.NewButton("切换槽位", func() {
		dialog.ShowInformation("提示", "切换槽位功能开发中...", window)
	})
	formatBtn := widget.NewButton("格式化", func() {
		dialog.ShowInformation("提示", "格式化功能开发中...", window)
	})

	rightPanel := container.NewVBox(
		widget.NewLabelWithStyle("设备状态", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		statusPanel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("设备操作", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		installDriverBtn,
		fastbootBtn,
		fastbootdBtn,
		bootSystemBtn,
		switchSlotBtn,
		formatBtn,
	)

	// 主布局：两列并排
	content := container.NewHBox(
		container.NewBorder(nil, nil, nil, nil, leftPanel),
		widget.NewSeparator(),
		container.NewBorder(nil, nil, nil, nil, rightPanel),
	)

	window.SetContent(content)

	// 初始检查设备状态
	updateDeviceStatus()

	return window
}

func main() {
	// 创建应用实例
	myApp := app.New()
	myWindow := myApp.NewWindow("用户登录")

	// 设置窗口大小
	myWindow.Resize(fyne.NewSize(400, 250))

	// 创建密码输入框
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("请输入密码")
	passwordEntry.Resize(fyne.NewSize(300, 30)) // 调整输入框大小

	// 创建登录按钮
	loginButton := widget.NewButton("登录", func() {
		password := passwordEntry.Text

		// 模拟登录验证
		if password == "123456" {
			dialog.ShowInformation("登录成功", "欢迎使用刷机工具！", myWindow)
			// 隐藏登录窗口
			myWindow.Hide()
			// 打开端口检查窗口
			portWindow := createPortCheckWindow(myApp)
			portWindow.Show()
		} else {
			dialog.ShowError(fmt.Errorf("密码错误"), myWindow)
		}
	})
	loginButton.Importance = widget.HighImportance

	// 创建重置按钮
	resetButton := widget.NewButton("重置", func() {
		passwordEntry.SetText("")
	})

	// 创建界面布局
	content := container.NewVBox(
		widget.NewLabelWithStyle("登录", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("密码:"),
		passwordEntry,
		container.NewHBox(
			widget.NewLabel(""),
			loginButton,
			resetButton,
		),
	)

	// 设置主内容
	myWindow.SetContent(container.NewCenter(content))

	// 显示窗口并运行
	myWindow.ShowAndRun()
}
