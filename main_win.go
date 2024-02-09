package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pirogom/walk"
	"github.com/pirogom/walkmgr"
)

type threadListData struct {
	WorkName   string
	doneCnt    int
	totalCnt   int
	errCnt     int
	processPer float64
}

func (t *threadListData) Init(name string, workCnt int) {
	t.totalCnt = workCnt
	t.WorkName = name
}

type MainWin struct {
	mgr *walkmgr.WalkUI

	srcEdit          *walk.LineEdit
	destEdit         *walk.LineEdit
	btnStart         *walk.PushButton
	replaceFileCheck *walk.CheckBox
	threadSlider     *walk.Slider
	bufSizeSlider    *walk.Slider

	btnSrcPath  *walk.PushButton
	btnDestPath *walk.PushButton

	listCtl *walkmgr.ListControl
	logList *walkmgr.ListControl

	threadListMt   []sync.Mutex
	threadList     []threadListData
	tlmt           sync.Mutex
	lastUpdateTime int64
}

/**
*	changeTarget
**/
func (w *MainWin) changeTarget() {
	w.btnDestPath.SetEnabled(false)
	w.btnStart.SetEnabled(false)
	defer func() {
		w.btnDestPath.SetEnabled(true)
		w.btnStart.SetEnabled(true)
	}()
	destPath, destPathErr := openPathDlg("복사 경로 선택")

	if destPathErr != nil {
		return
	}

	w.mgr.Sync(func() {
		w.destEdit.SetText(destPath)
	})
}

/**
*	changeSrc
**/
func (w *MainWin) changeSrc() {
	w.btnSrcPath.SetEnabled(false)
	defer w.btnSrcPath.SetEnabled(true)

	srcPath, srcPathErr := openPathDlg("원본 경로 선택")

	if srcPathErr != nil {
		return
	}

	w.mgr.Sync(func() {
		w.srcEdit.SetText(srcPath)
	})
}

/**
*	startCopy
**/
func (w *MainWin) startCopy() {
	if w.btnStart.Text() == "복사 시작" {
		srcPath := w.srcEdit.Text()
		destPath := w.destEdit.Text()
		//
		if srcPath == "" {
			walkmgr.MsgBox("원본 경로를 선택하십시요.")
			return
		}
		if destPath == "" {
			walkmgr.MsgBox("대상 경로를 선택하십시요.")
			return
		}
		//
		go w.CopyProc()

		w.SetEnable(false)
		w.btnStart.SetText("복사 중지")
	} else {
		w.SetEnable(true)
		w.btnStart.SetText("복사 시작")
	}
}

/**
*	ReplaceFileChecked
**/
func (w *MainWin) ReplaceFileChecked() {

}

/**
*	SetEnable
**/
func (w *MainWin) SetEnable(flag bool) {
	w.btnDestPath.SetEnabled(flag)
	w.btnSrcPath.SetEnabled(flag)
	w.replaceFileCheck.SetEnabled(flag)
	w.threadSlider.SetEnabled(flag)
	w.bufSizeSlider.SetEnabled(flag)
}

/**
*	ConvertDirList
**/
func (w *MainWin) ConvertDirList(dirList []string) []string {
	srcPath := w.srcEdit.Text()
	dstPath := w.destEdit.Text()

	outList := []string{}

	for _, v := range dirList {
		tp := strings.Replace(v, srcPath, "", -1)
		dp := filepath.Join(dstPath, tp)

		outList = append(outList, dp)
	}
	return outList
}

/**
*	GetThreadCount
**/
func (w *MainWin) GetThreadCount() int {
	tc := w.threadSlider.Value()
	if tc < 1 {
		return 1
	}
	return tc
}

/**
*	threadData
**/
type threadData struct {
	srcPath string
	dstPath string
}

/**
*	getSrcFileList
**/
func (w *MainWin) getSrcFileList(targetPath string) ([]string, [][]threadData) {
	mt := sync.Mutex{}

	srcPath := w.srcEdit.Text()
	dstPath := w.destEdit.Text()

	var dirCnt, fileCnt int
	dirList := []string{}

	tc := w.GetThreadCount()
	fileList := make([][]threadData, tc)
	var currThreadArray int

	var lastUpdateTime int64 = timeGetTime()

	mgr := walkmgr.NewFixed("파일/경로 분석중...", 500, 100)

	lb := mgr.LabelCenter("파일/경로 분석중...")

	mgr.Starting(func() {
		go func() {
			filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				if targetPath == path {
					return nil
				}

				currTime := timeGetTime()

				mt.Lock()

				//
				if info.IsDir() {
					tp := strings.Replace(path, srcPath, "", -1)
					dp := filepath.Join(dstPath, tp)

					dirList = append(dirList, dp)
					dirCnt++
				} else {

					td := threadData{}
					td.srcPath = path
					tp := strings.Replace(path, srcPath, "", -1)
					dp := filepath.Join(dstPath, tp)
					td.dstPath = dp

					if len(fileList) < currThreadArray+1 {
						fileList[currThreadArray] = []threadData{}
					}

					fileList[currThreadArray] = append(fileList[currThreadArray], td)
					fileCnt++

					currThreadArray++
					if currThreadArray >= tc {
						currThreadArray = 0
					}
				}
				//

				if lastUpdateTime+1000 < currTime {
					lastUpdateTime = currTime
					mgr.Sync(func() {
						lb.SetText(fmt.Sprintf("폴더 : %d 개 / 파일 : %d 개", dirCnt, fileCnt))
					})
				}
				mt.Unlock()
				return nil
			})
			mgr.Close()
		}()
	})

	mgr.IgnoreClosing()
	mgr.StartForeground()

	//
	w.listCtl.RemoveAll()

	//
	w.threadList = nil
	w.threadList = make([]threadListData, len(fileList)+1)
	w.threadListMt = make([]sync.Mutex, len(fileList)+1)
	//

	lastFileListIdx := -1
	for i := 0; i < len(fileList); i++ {
		if len(fileList[i]) == 0 {
			lastFileListIdx = i
			break
		}
	}

	if lastFileListIdx != -1 {
		fileList = fileList[:lastFileListIdx]
	}

	//
	w.threadList = nil
	w.threadList = make([]threadListData, len(fileList)+1)

	w.threadList[0].Init("폴더 생성", len(dirList))
	for i := 0; i < len(fileList); i++ {
		arrNum := i + 1
		w.threadList[arrNum].Init(fmt.Sprintf("%d번 스레드", arrNum), len(fileList[i]))
	}
	//
	w.InitThreadList()
	//

	return dirList, fileList
}

/**
*	InitThreadList
**/
func (w *MainWin) InitThreadList() {
	w.mgr.Sync(func() {
		for i := 0; i < len(w.threadList); i++ {
			w.threadListMt[i].Lock()
			nr := w.listCtl.AddItemData(fmt.Sprintf("%d", i))
			w.listCtl.SetItemData(nr, 1, w.threadList[i].WorkName)
			w.listCtl.SetItemData(nr, 2, fmt.Sprintf("%d", w.threadList[i].doneCnt))
			w.listCtl.SetItemData(nr, 3, fmt.Sprintf("%d", w.threadList[i].totalCnt))
			w.listCtl.SetItemData(nr, 4, fmt.Sprintf("%d", w.threadList[i].errCnt))
			w.listCtl.SetItemData(nr, 5, fmt.Sprintf("%0.2f%%", w.threadList[i].processPer))
			w.threadListMt[i].Unlock()
		}
		w.listCtl.UpdateAll()
	})
}

/**
*	UpdateThreadList
**/
func (w *MainWin) UpdateThreadList() {
	w.mgr.Sync(func() {
		for i := 0; i < len(w.threadList); i++ {
			w.threadListMt[i].Lock()
			w.listCtl.SetItemData(i, 1, w.threadList[i].WorkName)
			w.listCtl.SetItemData(i, 2, fmt.Sprintf("%d", w.threadList[i].doneCnt))
			w.listCtl.SetItemData(i, 3, fmt.Sprintf("%d", w.threadList[i].totalCnt))
			w.listCtl.SetItemData(i, 4, fmt.Sprintf("%d", w.threadList[i].errCnt))
			w.listCtl.SetItemData(i, 5, fmt.Sprintf("%0.2f%%", w.threadList[i].processPer))
			w.threadListMt[i].Unlock()
		}
	})
}

/**
*	UpdateListProc
**/
func (w *MainWin) UpdateListProc() {
	currTime := timeGetTime()

	if w.lastUpdateTime+1000 < currTime {
		w.tlmt.Lock()
		w.lastUpdateTime = currTime
		w.tlmt.Unlock()

		w.UpdateThreadList()
	}
}

/**
*	copyFile
**/
func (w *MainWin) copyFile(src string, dst string, overWrite bool, bufSize int64) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file.", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	if isExistFile(dst) {
		if overWrite {
			os.Remove(dst)
		} else {
			return fmt.Errorf("File %s already exists.", dst)
		}
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if err != nil {
		panic(err)
	}

	buf := make([]byte, bufSize)

	for {
		n, err := source.Read(buf)

		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return err
}

/**
*	CreateDir
**/
func (w *MainWin) CreateDir(dirList []string) {
	startTime := time.Now().Unix()
	w.Log("폴더 생성 시작.")

	for i := 0; i < len(dirList); i++ {
		hasErr := false

		if !isExistFile(dirList[i]) {
			if err := os.MkdirAll(dirList[i], 0644); err != nil {
				hasErr = true
			}
		}
		w.threadListMt[0].Lock()
		w.threadList[0].doneCnt++
		if hasErr {
			w.threadList[0].errCnt++
		}
		w.threadList[0].processPer = float64(w.threadList[0].doneCnt) / float64(w.threadList[0].totalCnt) * 100.0
		w.threadListMt[0].Unlock()
		w.UpdateListProc()
	}
	w.UpdateThreadList()
	w.Log("폴더 생성 완료. 소요시간: %d초", time.Now().Unix()-startTime)
}

/**
*	CopyFileList
**/
func (w *MainWin) CopyFileList(tn int, fileList []threadData, wg *sync.WaitGroup) {
	startTime := time.Now().Unix()
	w.Log("%d번 스레드 시작", tn)

	defer wg.Done()

	overWrite := w.replaceFileCheck.Checked()
	bufSize := w.bufSizeSlider.Value() * 1000000

	for i, _ := range fileList {
		if err := w.copyFile(fileList[i].srcPath, fileList[i].dstPath, overWrite, int64(bufSize)); err != nil {
			w.threadListMt[tn].Lock()
			w.threadList[tn].errCnt++
			w.threadList[tn].doneCnt++
			w.threadList[tn].processPer = float64(w.threadList[tn].doneCnt) / float64(w.threadList[tn].totalCnt) * 100.0
			w.threadListMt[tn].Unlock()
		} else {
			w.threadListMt[tn].Lock()
			w.threadList[tn].doneCnt++
			w.threadList[tn].processPer = float64(w.threadList[tn].doneCnt) / float64(w.threadList[tn].totalCnt) * 100.0
			w.threadListMt[tn].Unlock()
		}
		w.UpdateListProc()
	}
	w.Log("%d번 스레드 종료. 소요시간: %d초", tn, time.Now().Unix()-startTime)
}

/**
*	CopyProc
**/
func (w *MainWin) CopyProc() {

	startTime := time.Now().Unix()
	w.Log("파일복사를 시작합니다.")

	wg := sync.WaitGroup{}
	srcPath := w.srcEdit.Text()

	dirList, fileList := w.getSrcFileList(srcPath)

	// 디렉토리 먼저 생성
	if len(dirList) > 0 {
		w.CreateDir(dirList)
	}

	// 파일 복사 스레드 시작
	for i := 0; i < len(fileList); i++ {
		wg.Add(1)
		go w.CopyFileList(i+1, fileList[i], &wg)
	}
	wg.Wait()
	w.UpdateThreadList()

	w.mgr.Sync(func() {
		w.SetEnable(true)
		w.btnStart.SetText("복사 시작")
	})

	w.Log("파일복사 완료. 소요시간: %d초", time.Now().Unix()-startTime)
}

/**
*	AddLog
**/
func (w *MainWin) Log(format string, a ...any) {
	w.mgr.Sync(func() {
		w.logList.AddItemData(fmt.Sprintf(format, a...))
	})
}

/**
*	NewMainWin
**/
func NewMainWin() {
	cpuCount := runtime.NumCPU()

	nw := MainWin{}

	nw.mgr = walkmgr.NewWin("모두의 복사기 v"+MOCOPY_VER, 800, 600)
	nw.mgr.DisableMaxBox()
	//nw.mgr.NoResize()

	nw.mgr.HComposite()
	nw.mgr.Label("원본 경로:")
	nw.srcEdit = nw.mgr.LineStatic("")
	nw.btnSrcPath = nw.mgr.PushButton("원본 경로 선택", nw.changeSrc)
	nw.mgr.End()

	nw.mgr.HComposite()
	nw.mgr.Label("대상 경로:")
	nw.destEdit = nw.mgr.LineStatic("")
	nw.btnDestPath = nw.mgr.PushButton("대상 경로 선택", nw.changeTarget)
	nw.mgr.End()

	nw.mgr.HGroupBox("옵션")
	nw.mgr.Label("복사에 사용할 스레드수:")
	nw.threadSlider = nw.mgr.Slider(1, cpuCount*6, cpuCount)
	nw.mgr.Label("버퍼 크기(MB):")
	nw.bufSizeSlider = nw.mgr.Slider(1, 16, 3)
	nw.replaceFileCheck = nw.mgr.CheckBox("중복 파일 덮어쓰기", false, nw.ReplaceFileChecked)
	nw.mgr.End()

	nw.listCtl = walkmgr.NewListControl(nw.mgr, nil)

	nw.listCtl.AddColumn("#", 50, walkmgr.LISTCTRL_ALIGN_LEFT, false)
	nw.listCtl.AddColumn("작업", 80, walkmgr.LISTCTRL_ALIGN_CENTER, false)
	nw.listCtl.AddColumn("완료", 80, walkmgr.LISTCTRL_ALIGN_RIGHT, false)
	nw.listCtl.AddColumn("전체", 80, walkmgr.LISTCTRL_ALIGN_RIGHT, false)
	nw.listCtl.AddColumn("오류", 80, walkmgr.LISTCTRL_ALIGN_RIGHT, false)
	nw.listCtl.AddColumn("진행률", 80, walkmgr.LISTCTRL_ALIGN_CENTER, false)
	nw.listCtl.Create()

	nw.logList = walkmgr.NewListControl(nw.mgr, nil)
	nw.logList.AddColumn("로그", 700, walkmgr.LISTCTRL_ALIGN_LEFT, false)
	nw.logList.Create()
	nw.logList.FixHeight(200)

	nw.btnStart = nw.mgr.PushButton("복사 시작", nw.startCopy)
	nw.mgr.VSpacer()

	nw.mgr.StartForeground()
}
