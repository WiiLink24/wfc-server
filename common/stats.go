package common

var OnlineUsers int

func init(){ 
    OnlineUsers = 0 
}

func OnlineStatUpdate(t int) {
    OnlineUsers += t
}

