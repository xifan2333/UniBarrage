package trace

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// 日志缓冲区
var logBuffer []string

var logLevel = 0

const maxLogLines = 20 // 固定显示高度

// 样式定义：软件名和版本，日志级别、日期和时间
var (
	headerStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#FF69B4")).
			Padding(2, 2). // 上下和左右的间距
			Bold(true).
			Width(82).
			Align(lipgloss.Center) // 居中对齐

	titleStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#6D00E8")).
			Padding(0, 1).
			Bold(true)

	infoStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#339933")).
			Padding(0, 1).
			Bold(true)

	warnStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#FFCC33")).
			Padding(0, 1).
			Bold(true)

	errorStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#CC3333")).
			Padding(0, 1).
			Bold(true)

	timeStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	messageStyle = lipgloss.
			NewStyle().
			Padding(0, 1)

	// 抖音
	douyinStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1).
			Bold(true)

	// 哔哩哔哩粉
	bilibiliStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#FF69B4")).
			Padding(0, 1).
			Bold(true)

	// 快手橙
	kuaishouStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#FF6600")).
			Padding(0, 1).
			Bold(true)

	// 虎牙黄
	huyaStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Padding(0, 1).
			Bold(true)

	// 斗鱼灰
	douyuStyle = lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#A9A9A9")).
			Padding(0, 1).
			Bold(true)

	// 小红书红
	xiaohongshuStyle = lipgloss.
				NewStyle().
				Foreground(lipgloss.Color("#FF2442")).
				Padding(0, 1).
				Bold(true)
)

// printHeader 打印程序启动的欢迎信息
func printHeader() {
	header := headerStyle.Render(fmt.Sprintf("🎉 UniBarrage v%s 🎉", "1.0.0"))
	_, _ = fmt.Fprint(os.Stderr, "\033[H\033[2J") // 清空屏幕
	_, _ = fmt.Fprint(os.Stderr, "\033[?25l")    // 隐藏光标
	_, _ = fmt.Fprintln(os.Stderr, "\n"+header)
}

// 添加日志到缓冲区，并控制缓冲区大小
func addLogToBuffer(log string) {
	logBuffer = append(logBuffer, log)
	if len(logBuffer) > maxLogLines {
		logBuffer = logBuffer[1:] // 移除最早的一条日志
	}
}

// 刷新日志区域（固定在 header 下方）
func refreshLogArea() {
	_, _ = fmt.Fprint(os.Stderr, "\033[7;0H\033[J") // 将光标移动到第 7 行，并清除其后的内容
	for _, log := range logBuffer {
		_, _ = fmt.Fprintln(os.Stderr, log)
	}
	_, _ = fmt.Fprint(os.Stderr, "\033[0m")    // 重置所有样式，避免背景色残留
	_, _ = fmt.Fprint(os.Stderr, "\033[999;0H") // 将光标移到屏幕外（第 999 行）
}

// Print 输出函数：根据级别美化日志并输出
func Print(level, msg string) {
	level = strings.ToUpper(level)

	if logLevel == 0 {
		// 使用美化输出
		now := time.Now()
		timestamp := timeStyle.Render(now.Format("15:04:05"))
		title := titleStyle.Render("[ UniBarrage ]")
		message := messageStyle.Render(msg)

		var levelStyled string
		switch level {
		case "INFO":
			levelStyled = infoStyle.Render(" INFO ")
		case "WARN":
			levelStyled = warnStyle.Render(" WARN ")
		case "ERROR":
			levelStyled = errorStyle.Render(" ERRO ")
		case "DOUYIN":
			levelStyled = douyinStyle.Render(" DOUY ")
		case "BILIBILI":
			levelStyled = bilibiliStyle.Render(" BILI ")
		case "KUAISHOU":
			levelStyled = kuaishouStyle.Render(" KUAI ")
		case "HUYA":
			levelStyled = huyaStyle.Render(" HUYA ")
		case "DOUYU":
			levelStyled = douyuStyle.Render(" DOUV ")
		case "XIAOHONGSHU", "XHS":
			levelStyled = xiaohongshuStyle.Render(" XHS  ")
		default:
			levelStyled = level
		}

		finalMessage := fmt.Sprintf("%s%s%s%s", title, timestamp, levelStyled, message)
		addLogToBuffer(finalMessage)
		refreshLogArea()
	} else if logLevel == 1 || (logLevel >= 2 && (level == "WARN" || level == "ERROR")) {
		// 原始输出格式
		fmt.Printf("[%s] %s %s\n", level, time.Now().Format("15:04:05"), msg)
	}
}

// Printf 根据级别美化日志并输出，支持格式化字符串
func Printf(level, format string, a ...interface{}) {
	level = strings.ToUpper(level)

	if logLevel == 0 {
		// 使用美化输出
		now := time.Now()
		timestamp := timeStyle.Render(now.Format("15:04:05"))
		title := titleStyle.Render("[ UniBarrage ]")
		message := messageStyle.Render(fmt.Sprintf(format, a...))

		var levelStyled string
		switch level {
		case "INFO":
			levelStyled = infoStyle.Render(" INFO ")
		case "WARN":
			levelStyled = warnStyle.Render(" WARN ")
		case "ERROR":
			levelStyled = errorStyle.Render(" ERRO ")
		case "DOUYIN":
			levelStyled = douyinStyle.Render(" DOUY ")
		case "BILIBILI":
			levelStyled = bilibiliStyle.Render(" BILI ")
		case "KUAISHOU":
			levelStyled = kuaishouStyle.Render(" KUAI ")
		case "HUYA":
			levelStyled = huyaStyle.Render(" HUYA ")
		case "DOUYU":
			levelStyled = douyuStyle.Render(" DOUV ")
		case "XIAOHONGSHU", "XHS":
			levelStyled = xiaohongshuStyle.Render(" XHS  ")
		default:
			levelStyled = level
		}

		finalMessage := fmt.Sprintf("%s%s%s%s", title, timestamp, levelStyled, message)
		addLogToBuffer(finalMessage)
		refreshLogArea()
	} else if logLevel == 1 || (logLevel >= 2 && (level == "WARN" || level == "ERROR")) {
		// 原始输出格式
		fmt.Printf("[%s] %s ", level, time.Now().Format("15:04:05"))
		fmt.Printf(format, a...)
		fmt.Println()
	}
}

// HandleSignal 捕获系统信号并处理
func HandleSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	Printf("INFO", "收到信号: %v, 正在终止程序...", sig)

	// 显示光标并终止整个程序
	_, _ = fmt.Fprint(os.Stderr, "\033[?25h") // 显示光标
	os.Exit(1)
}

func Init(level int) {
	if level == 0 {
		printHeader()
	}

	if level == 1 {
		logLevel = 1
	}

	if level >= 2 {
		logLevel = 2
	}
}
