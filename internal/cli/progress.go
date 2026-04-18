package cli

import (
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	"log"
	"strings"
	"sync"
	"time"
)

var progressReminderInterval = 10 * time.Second

type endpointMatcher func(string) bool

type collectedDataOperationSpec struct {
	matches        endpointMatcher
	startMessage   string
	reminderPrefix string
	successMessage func(clinicapi.RequestResult) string
	failureMessage func(clinicapi.RequestFailure) string
}

type progressTicker struct {
	stop      chan struct{}
	startedAt time.Time
}

type collectedDataProgress struct {
	logger *log.Logger

	operation collectedDataOperationSpec

	mu                    sync.Mutex
	printedSelect         bool
	printedStatus         bool
	printedRebuildTrigger bool
	printedRebuilding     bool
	printedOperationStart bool
	rebuildTicker         progressTicker
	operationTicker       progressTicker
}

func newCollectedDataProgress(logger *log.Logger, operation collectedDataOperationSpec) *collectedDataProgress {
	if logger == nil {
		logger = log.New(nilWriter{}, "", 0)
	}
	return &collectedDataProgress{
		logger:    logger,
		operation: operation,
	}
}

func newDataDownloadProgress(logger *log.Logger) *collectedDataProgress {
	return newCollectedDataProgress(logger, collectedDataOperationSpec{
		matches: func(endpoint string) bool {
			return strings.Contains(endpoint, "/download/")
		},
		startMessage:   "正在下载 bundle...",
		reminderPrefix: "正在下载 bundle...",
		successMessage: func(result clinicapi.RequestResult) string {
			return fmt.Sprintf("bundle 下载完成（%s，%s）", humanSize(result.ResponseBytes), formatElapsed(result.Duration))
		},
		failureMessage: func(failure clinicapi.RequestFailure) string {
			return timedMessage("bundle 下载失败", failure.Duration)
		},
	})
}

func newRetainedSlowQueriesProgress(logger *log.Logger) *collectedDataProgress {
	return newCollectedDataProgress(logger, collectedDataOperationSpec{
		matches: func(endpoint string) bool {
			return strings.HasSuffix(endpoint, "/slowqueries")
		},
		startMessage:   "正在查询 slow query 记录...",
		reminderPrefix: "正在查询 slow query 记录...",
		successMessage: func(result clinicapi.RequestResult) string {
			return timedMessage("slow query 查询完成", result.Duration)
		},
		failureMessage: func(failure clinicapi.RequestFailure) string {
			return timedMessage("slow query 查询失败", failure.Duration)
		},
	})
}

func (p *collectedDataProgress) Hooks() clinicapi.Hooks {
	return clinicapi.Hooks{
		OnRequestStart: p.onRequestStart,
		OnRequestDone:  p.onRequestDone,
		OnError:        p.onRequestError,
	}
}

func (p *collectedDataProgress) Close() {
	p.stopRebuildTicker()
	p.stopOperationTicker()
}

func (p *collectedDataProgress) onRequestStart(info clinicapi.RequestInfo) {
	switch {
	case strings.HasSuffix(info.Endpoint, "/data"):
		p.mu.Lock()
		shouldPrint := !p.printedSelect
		if shouldPrint {
			p.printedSelect = true
		}
		p.mu.Unlock()
		if shouldPrint {
			p.print("正在选择 collected data bundle...")
		}
	case strings.HasSuffix(info.Endpoint, "/data_status"):
		p.mu.Lock()
		shouldPrint := !p.printedStatus
		if shouldPrint {
			p.printedStatus = true
		}
		p.mu.Unlock()
		if shouldPrint {
			p.print("正在检查 bundle 状态...")
		}
	case strings.HasSuffix(info.Endpoint, "/rebuild"):
		p.mu.Lock()
		shouldPrint := !p.printedRebuildTrigger
		if shouldPrint {
			p.printedRebuildTrigger = true
		}
		p.mu.Unlock()
		if shouldPrint {
			p.print("bundle 需要 rebuild，正在触发 rebuild...")
		}
	case p.matchesOperation(info.Endpoint):
		if stopped, elapsed := p.stopRebuildTicker(); stopped {
			p.print(fmt.Sprintf("bundle 已就绪（等待 %s）。", formatElapsed(elapsed)))
		}
		p.mu.Lock()
		shouldPrint := !p.printedOperationStart
		if shouldPrint {
			p.printedOperationStart = true
		}
		p.mu.Unlock()
		if shouldPrint {
			p.print(p.operation.startMessage)
		}
		p.startOperationTicker()
	}
}

func (p *collectedDataProgress) onRequestDone(result clinicapi.RequestResult) {
	switch {
	case strings.HasSuffix(result.Endpoint, "/rebuild"):
		p.print("正在等待 rebuild 完成...")
		p.startRebuildTicker()
	case p.matchesOperation(result.Endpoint):
		p.stopOperationTicker()
		if p.operation.successMessage != nil {
			p.print(p.operation.successMessage(result))
		}
	}
}

func (p *collectedDataProgress) onRequestError(failure clinicapi.RequestFailure) {
	switch {
	case p.matchesOperation(failure.Endpoint):
		if isSlowQueryProcessingFailure(failure.Err) {
			p.mu.Lock()
			shouldPrint := !p.printedRebuilding
			if shouldPrint {
				p.printedRebuilding = true
			}
			p.mu.Unlock()
			if shouldPrint {
				p.print("slow query 正在 rebuilding，等待完成...")
			}
			return
		}
		p.stopOperationTicker()
		if p.operation.failureMessage != nil {
			p.print(p.operation.failureMessage(failure))
		}
	case strings.HasSuffix(failure.Endpoint, "/rebuild"), strings.HasSuffix(failure.Endpoint, "/data_status"):
		if stopped, elapsed := p.stopRebuildTicker(); stopped {
			p.print(fmt.Sprintf("等待 rebuild 失败（%s）。", formatElapsed(elapsed)))
		} else if strings.HasSuffix(failure.Endpoint, "/data_status") {
			p.print(timedMessage("检查 bundle 状态失败", failure.Duration))
		}
	case strings.HasSuffix(failure.Endpoint, "/data"):
		p.print(timedMessage("选择 collected data bundle 失败", failure.Duration))
	}
}

func (p *collectedDataProgress) startRebuildTicker() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rebuildTicker.stop != nil {
		return
	}
	stop := make(chan struct{})
	p.rebuildTicker = progressTicker{
		stop:      stop,
		startedAt: time.Now(),
	}
	go p.runTicker(stop, p.rebuildTicker.startedAt, "正在等待 rebuild 完成...")
}

func (p *collectedDataProgress) stopRebuildTicker() (bool, time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rebuildTicker.stop == nil {
		return false, 0
	}
	elapsed := time.Since(p.rebuildTicker.startedAt)
	close(p.rebuildTicker.stop)
	p.rebuildTicker = progressTicker{}
	return true, elapsed
}

func (p *collectedDataProgress) startOperationTicker() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.operationTicker.stop != nil {
		return
	}
	stop := make(chan struct{})
	p.operationTicker = progressTicker{
		stop:      stop,
		startedAt: time.Now(),
	}
	go p.runTicker(stop, p.operationTicker.startedAt, p.operation.reminderPrefix)
}

func (p *collectedDataProgress) stopOperationTicker() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.operationTicker.stop == nil {
		return false
	}
	close(p.operationTicker.stop)
	p.operationTicker = progressTicker{}
	return true
}

func (p *collectedDataProgress) runTicker(stop <-chan struct{}, startedAt time.Time, prefix string) {
	ticker := time.NewTicker(progressReminderInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			p.print(fmt.Sprintf("%s 已耗时 %s", prefix, time.Since(startedAt).Round(time.Second)))
		}
	}
}

func (p *collectedDataProgress) matchesOperation(endpoint string) bool {
	return p != nil && p.operation.matches != nil && p.operation.matches(endpoint)
}

func (p *collectedDataProgress) print(line string) {
	if p == nil || p.logger == nil || strings.TrimSpace(line) == "" {
		return
	}
	p.logger.Println(line)
}

func isSlowQueryProcessingFailure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "the log is processing")
}

func humanSize(bytes int) string {
	value := float64(bytes)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := units[0]
	for i := 1; i < len(units) && value >= 1024; i++ {
		value /= 1024
		unit = units[i]
	}
	switch unit {
	case "B":
		return fmt.Sprintf("%d %s", bytes, unit)
	default:
		return fmt.Sprintf("%.1f %s", value, unit)
	}
}

func timedMessage(prefix string, elapsed time.Duration) string {
	return fmt.Sprintf("%s（%s）", strings.TrimSpace(prefix), formatElapsed(elapsed))
}

func formatElapsed(elapsed time.Duration) string {
	if elapsed <= 0 {
		return "0s"
	}
	round := time.Millisecond
	switch {
	case elapsed >= 10*time.Second:
		round = time.Second
	case elapsed >= time.Second:
		round = 100 * time.Millisecond
	}
	return elapsed.Round(round).String()
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
