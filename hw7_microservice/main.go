package main

// import (
// 	"context"
// 	"testing"
// )

// const (
// 	// какой адрес-порт слушать серверу
// 	listenAddr string = "127.0.0.1:8082"

// 	// кого по каким методам пускать
// 	ACLData string = `{
// 	"logger":    ["/main.Admin/Logging"],
// 	"stat":      ["/main.Admin/Statistics"],
// 	"biz_user":  ["/main.Biz/Check", "/main.Biz/Add"],
// 	"biz_admin": ["/main.Biz/*"]
// }`
// )

// утилитарная функция для коннекта к серверу
// func getGrpcConn(t *testing.T) *grpc.ClientConn {
// 	grcpConn, err := grpc.Dial(
// 		listenAddr,
// 		grpc.WithInsecure(),
// 	)
// 	if err != nil {
// 		t.Fatalf("cant connect to grpc: %v", err)
// 	}
// 	return grcpConn
// }

// // получаем контекст с нужнымы метаданными для ACL
// func getConsumerCtx(consumerName string) context.Context {
// 	// ctx, _ := context.WithTimeout(context.Background(), time.Second)
// 	ctx := context.Background()
// 	md := metadata.Pairs(
// 		"consumer", consumerName,
// 	)
// 	return metadata.NewOutgoingContext(ctx, md)
// }

func main() {
	// ctx, finish := context.WithCancel(context.Background())
	// err := StartMyMicroservice(ctx, listenAddr, ACLData)
	// if err != nil {
	// 	ftm.Printf("cant start server initial: %v", err)
	// }
	// wait(1)
	// defer func() {
	// 	finish()
	// 	wait(1)
	// }()

	// conn := getGrpcConn(t)
	// defer conn.Close()
	// biz := NewBizClient(conn)
	// adm := NewAdminClient(conn)
	// logStream1, err := adm.Logging(getConsumerCtx("logger"), &Nothing{})

	// select {
	// case evt, err <- logStream1.Recv():
	// 	fmt.Printf("%v", evt)
	// }

}
