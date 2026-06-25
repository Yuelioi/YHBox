// Package window 提供窗口类别节点: GetWindow(解析→Window 对象), 以及窗口控制
// WindowState / MoveResizeWindow / CloseWindow(后续任务进同包)。
//
// 注: Category=="Window" 的节点不止本包 —— WindowTarget / WaitWindow / WaitWindowGone 在
// internal/nodes/system/, BringWindowForeground 在 internal/nodes/input/(D1 起 Category 标 "Window")。
// 找全部窗口节点按 Category 而非包。
package window
