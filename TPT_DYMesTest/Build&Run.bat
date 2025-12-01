@echo off
chcp 65001 >nul
echo ========================================
echo TPT MES 測試伺服器啟動腳本 (強制重新編譯)
echo ========================================
echo.

REM ---------------編譯檔案------------------
echo [狀態] 正在編譯程式 (go build -o DYMesTest.exe)...
echo.
go build -o DYMesTest.exe

if errorlevel 1 (
    echo.
    echo [錯誤] 編譯失敗！
    echo 請檢查 Go 環境是否正確安裝
    pause
    exit /b 1
)

echo.
echo [成功] 編譯完成！
echo.
REM --------------------------------------


REM ---------------啟動伺服器---------------
echo 啟動伺服器...
echo.
echo Web 介面: http://localhost:5179
echo TCP 埠號: 50200
echo.
echo 按 Ctrl+C 停止伺服器
echo ========================================
echo.
REM ---------------啟動伺服器---------------


DYMesTest.exe

pause