# Client 

Client to talk to hal

## Usage

```zsh
go build -o status_update main.go
POST_URL=https://localhost:8000/update AUTH_TOKEN=your-secret-token ./status_update
```

And then submit your messages...

![alt text](images/client.png)


