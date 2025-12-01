@echo off
chcp 65001
echo ========================================
echo TPT MES 測試伺服器啟動腳本
echo ========================================
echo.

REM 檢查是否已編譯
if not exist "DYMesTest.exe" (
    echo 正在編譯程式...
    go build -o DYMesTest.exe
    if errorlevel 1 (
        echo 編譯失敗！
        pause
        exit /b 1
    )
    echo 編譯成功！
    echo.
)

echo 啟動伺服器...
echo.
echo Web 介面: http://localhost:8080
echo TCP 埠號: 50200
echo.
echo 按 Ctrl+C 停止伺服器
echo ========================================
echo.

DYMesTest.exe

pause

