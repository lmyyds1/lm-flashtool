package main

import (
	"fmt"
	"net"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// checkPort 检查指定端口是否被占用
func checkPort(port int) (bool, string) {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		// 端口被占用
		return true, fmt.Sprintf("端口 %d 已被占用", port)
	}
	listener.Close()
	return false, fmt.Sprintf("端口 %d 可用", port)
}

// createPortCheckWindow 创建端口检查窗口
func createPortCheckWindow(app fyne.App) fyne.Window {
	window := app.NewWindow("端口检查工具")
	window.Resize(fyne.NewSize(400, 200))

	// 结果显示标签
	resultLabel := widget.NewLabel("点击按钮检查端口占用情况")

	// 检查端口按钮
	checkButton := widget.NewButton("检查5037端口占用", func() {
		occupied, msg := checkPort(5037)
		if occupied {
			resultLabel.SetText(msg + "\n\n提示: 可能是 ADB 服务正在运行")
		} else {
			resultLabel.SetText(msg)
		}
	})
	checkButton.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabelWithStyle("端口检查工具", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		checkButton,
		widget.NewSeparator(),
		resultLabel,
	)

	window.SetContent(container.NewCenter(content))
	return window
}

func main() {
	// 创建应用实例
	myApp := app.New()
	myWindow := myApp.NewWindow("用户登录")

	// 设置窗口大小
	myWindow.Resize(fyne.NewSize(400, 300))

	// 创建用户名输入框
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("请输入用户名")

	// 创建密码输入框
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("请输入密码")

	// 创建登录按钮
	loginButton := widget.NewButton("登录", func() {
		username := usernameEntry.Text
		password := passwordEntry.Text

		// 模拟登录验证
		if username == "admin" && password == "123456" {
			dialog.ShowInformation("登录成功", fmt.Sprintf("欢迎, %s!", username), myWindow)
			// 登录成功后打开端口检查窗口
			portWindow := createPortCheckWindow(myApp)
			portWindow.Show()
		} else {
			dialog.ShowError(fmt.Errorf("用户名或密码错误"), myWindow)
		}
	})
	loginButton.Importance = widget.HighImportance

	// 创建重置按钮
	resetButton := widget.NewButton("重置", func() {
		usernameEntry.SetText("")
		passwordEntry.SetText("")
	})

	// 创建界面布局
	content := container.NewVBox(
		widget.NewLabelWithStyle("用户登录", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("用户名:"),
		usernameEntry,
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
