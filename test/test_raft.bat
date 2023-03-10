cd ./../
for %%i in (1 2 3 ) do (
    echo %%i
    timeout /t 2 /nobreak > nul
    start go test -v -race -run=Raft%%i
)

