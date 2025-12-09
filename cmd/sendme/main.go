package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"alt-sendme-go/internal/receiver"
	"alt-sendme-go/internal/sender"
	"alt-sendme-go/internal/ticket"
	"alt-sendme-go/internal/utils"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	port       int
	outputPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sendme",
		Short: "P2P file transfer tool",
		Long:  "A simple peer-to-peer file transfer tool built with Go and libp2p",
	}

	sendCmd := &cobra.Command{
		Use:   "send <file>",
		Short: "Send a file",
		Long:  "Start sharing a file and generate a ticket for the receiver",
		Args:  cobra.ExactArgs(1),
		RunE:  runSend,
	}
	sendCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on (0 = random)")

	receiveCmd := &cobra.Command{
		Use:   "receive <ticket>",
		Short: "Receive a file",
		Long:  "Receive a file using a ticket from the sender",
		Args:  cobra.ExactArgs(1),
		RunE:  runReceive,
	}
	receiveCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path (directory or file path)")
	receiveCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on (0 = random)")

	rootCmd.AddCommand(sendCmd, receiveCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSend(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// 创建进度条
	var bar *progressbar.ProgressBar
	var fileSize int64

	onProgress := func(bytesTransferred, totalBytes int64, speed float64) {
		if bar == nil {
			fileSize = totalBytes
			bar = progressbar.DefaultBytes(
				totalBytes,
				"Transferring",
			)
		}
		bar.Set64(bytesTransferred)
	}

	// 启动分享
	tkt, err := sender.StartShare(ctx, filePath, port, onProgress)
	if err != nil {
		return fmt.Errorf("failed to start sharing: %w", err)
	}

	// 编码 ticket
	encodedTicket, err := tkt.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode ticket: %w", err)
	}

	fmt.Printf("\nFile sharing started!\n")
	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("Size: %s\n", utils.FormatFileSize(fileSize))
	fmt.Printf("\nShare this ticket with the receiver:\n\n")
	fmt.Printf("%s\n\n", encodedTicket)
	fmt.Println("Waiting for receiver to connect...")
	fmt.Println("Press Ctrl+C to stop sharing")

	// 等待直到上下文取消
	<-ctx.Done()

	// 清理资源会在 defer 中处理
	return nil
}

func runReceive(cmd *cobra.Command, args []string) error {
	encodedTicket := args[0]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nCancelling...")
		cancel()
	}()

	// 解码 ticket
	tkt, err := ticket.Decode(encodedTicket)
	if err != nil {
		return fmt.Errorf("failed to decode ticket: %w", err)
	}

	fmt.Printf("Receiving file: %s\n", tkt.FileName)
	fmt.Printf("Size: %s\n", utils.FormatFileSize(tkt.FileSize))
	fmt.Println()

	// 创建进度条
	var bar *progressbar.ProgressBar

	onProgress := func(bytesTransferred, totalBytes int64, speed float64) {
		if bar == nil {
			bar = progressbar.DefaultBytes(
				totalBytes,
				"Downloading",
			)
		}
		bar.Set64(bytesTransferred)
	}

	// 接收文件
	if err := receiver.Receive(ctx, tkt, outputPath, port, onProgress); err != nil {
		return fmt.Errorf("failed to receive file: %w", err)
	}

	if bar != nil {
		bar.Finish()
	}

	fmt.Println("\nFile received successfully!")
	return nil
}


