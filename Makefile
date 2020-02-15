build:
	cd cmd && GOOS=linux GOARCH=arm GOARM=5 go build -o character

deploy: build
	scp ./cmd/character pi@192.168.0.52:/home/pi/character
