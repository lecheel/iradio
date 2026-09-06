#! /bin/bash
echo "build iradio"
go build -o iradio main.go

echo "build iradio-client"
go build -o iradio-client ./client/

# cp iradio ~/bin
# cp iradio-client ~/bin


