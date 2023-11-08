package sys

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Restarter is the interface that wraps the Restart method.
type Restarter interface {
	Restart(executableName string) error
}

// GetLocalIP 返回本地机器的内网IPv4地址
func GetLocalIP() (string, error) {
	var preferredIP, fallbackIP string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				// 优先选择以"192"开头的地址
				if preferredIP == "" && ip[:3] == "192" {
					preferredIP = ip
				} else if fallbackIP == "" && ip[:2] == "10" {
					// 如果还没有找到"192"，则选择以"10"开头的地址
					fallbackIP = ip
				} else if fallbackIP == "" {
					// 检查是否在"172.16.0.0"到"172.31.255.255"之间
					if ipnet.IP.To4()[0] == 172 && ipnet.IP.To4()[1] >= 16 && ipnet.IP.To4()[1] <= 31 {
						fallbackIP = ip
					}
				}
			}
		}
	}
	// 根据优先级返回IP地址
	if preferredIP != "" {
		return preferredIP, nil
	}
	if fallbackIP != "" {
		return fallbackIP, nil
	}
	return "", fmt.Errorf("cannot find local IP address")
}

func GetPublicIP() (string, error) {
	// 访问iframe提供的URL
	resp, err := http.Get("http://only-985281-116-238-216-144.nstool.yqkk.link/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 确认HTTP请求成功了
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not fetch iframe data: HTTP status %d", resp.StatusCode)
	}

	// 读取响应体
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 将响应体转换为字符串
	body := string(b)

	// 使用正则表达式查找IP地址
	re := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	ipMatches := re.FindAllString(body, -1)

	if ipMatches == nil {
		return "", fmt.Errorf("no IP address found")
	}

	// 第一个匹配的就是公共IP
	publicIP := ipMatches[0]

	// 返回找到的公共IP地址
	return publicIP, nil
}

// findIFrameSrc 遍历HTML节点以找到iframe标签的src属性
func findIFrameSrc(n *html.Node) (string, bool) {
	if n.Type == html.ElementNode && n.Data == "iframe" {
		for _, a := range n.Attr {
			if a.Key == "src" {
				return a.Val, true
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if src, found := findIFrameSrc(c); found {
			return src, true
		}
	}
	return "", false
}

// GetExecutableName 返回当前执行文件的名称
func GetExecutableName() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(executable, filepath.Ext(executable)), nil
}

// linux
func setConsoleTitleLinux(title string) {
	fmt.Printf("\033]0;%s\007", title)
}

// InitBase 解析参数并检测
//
//	如果在 windows 下双击打开了程序，程序将在此函数释出脚本后终止；
//	如果传入 -h 参数，程序将打印帮助后终止；
//	如果传入 -d 参数，程序将在启动 daemon 后终止。
func InitBase() {
	if runtime.GOOS == "windows" {
		if RunningByDoubleClick() {
			err := NoMoreDoubleClick()
			if err != nil {
				log.Printf("遇到错误: %v", err)
				time.Sleep(time.Second * 5)
			}
			os.Exit(0)
		}
	} else {
		fmt.Printf("InitBase function is not implemented for %s\n", runtime.GOOS)
	}
}

// 判断系统
func setConsoleTitle(title string) error {
	switch runtime.GOOS {
	case "windows":
		return setConsoleTitleWindows(title)
	case "linux":
		setConsoleTitleLinux(title)
	default:
		fmt.Fprintf(os.Stderr, "setConsoleTitle not implemented for %s\n", runtime.GOOS)
	}
	return nil
}

// SetTitle sets the window title to "Gensokyo © 2023 - [Year] Hoshinonyaruko".
func SetTitle(title string) {
	err := setConsoleTitle(title)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set title: %v\n", err)
	}
}

// RestartApplication 封装了应用程序的重启逻辑
func RestartApplication() {
	execName, err := GetExecutableName() // 确保这个函数返回正确
	if err != nil {
		log.Println("Error getting executable name:", err)
		os.Exit(1) // 出错时退出码不为0
	}

	restarter := NewRestarter()
	if err := restarter.Restart(execName); err != nil {
		log.Println("Error restarting application:", err)
		os.Exit(1) // 出错时退出码不为0
	}

	// 创建restart.flag文件，表示自己正在restart
	if _, err := os.Create("restart.flag"); err != nil {
		log.Println("Unable to create restart flag:", err)
		os.Exit(1) // 出错时退出码不为0
	}

	// 退出程序
	os.Exit(0)
}

func findIP(node *html.Node) (string, bool) {
	if node.Type == html.ElementNode && node.Data == "iframe" {
		for _, a := range node.Attr {
			if a.Key == "src" {
				// We found the iframe, now let's send a request to the src URL
				resp, err := http.Get(a.Val)
				if err != nil {
					return "", false
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return "", false
				}
				doc, err := html.Parse(resp.Body)
				if err != nil {
					return "", false
				}
				// The actual IP might be in a <div> or <span>, depending on the page structure
				var f func(*html.Node) string
				f = func(n *html.Node) string {
					if n.Type == html.TextNode && strings.HasPrefix(n.Data, "您的IP地址信息:") {
						return strings.TrimSpace(strings.Split(n.Data, " ")[1])
					}
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if result := f(c); result != "" {
							return result
						}
					}
					return ""
				}
				ip := f(doc)
				if ip != "" {
					return ip, true
				}
			}
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if ip, found := findIP(c); found {
			return ip, true
		}
	}
	return "", false
}
