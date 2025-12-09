package utils

import (
	"fmt"
	"time"
)

// ProgressCallback 是进度回调函数类型
type ProgressCallback func(bytesTransferred, totalBytes int64, speed float64)

// CalculateSpeed 计算传输速度（字节/秒）
func CalculateSpeed(bytesTransferred int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(bytesTransferred) / elapsed.Seconds()
}

// FormatSpeed 格式化速度显示
func FormatSpeed(speed float64) string {
	const unit = 1024.0
	if speed < unit {
		return fmt.Sprintf("%.2f B/s", speed)
	}
	div, exp := unit, 0
	for n := speed / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB/s", speed/div, "KMGTPE"[exp])
}

// FormatProgress 格式化进度百分比
func FormatProgress(bytesTransferred, totalBytes int64) string {
	if totalBytes == 0 {
		return "0.00%"
	}
	percentage := float64(bytesTransferred) / float64(totalBytes) * 100
	return fmt.Sprintf("%.2f%%", percentage)
}


