As example, you have camera:
```
rtsp://admin:password@192.168.1.123:554/cam/realmonitor?channel=1&subtype=0
````

Run [Ngrok](https://ngrok.com/) on any computer in you LAN (use [your token](https://dashboard.ngrok.com/get-started/your-authtoken)):
```
ngrok tcp 192.168.1.123:554 --authtoken eW91IHNoYWxsIG5vdCBwYXNzCnlvdSBzaGFsbCBub3QgcGFzcw
```

You will get similar output:
```
tcp://0.tcp.eu.ngrok.io:11465 -> 192.168.1.123:554
```

Now you have working stream:
```
rtsp://admin:password@0.tcp.eu.ngrok.io:11465/cam/realmonitor?channel=1&subtype=0
```
