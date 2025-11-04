package main

func main() {
	// fileID := flag.String("id", "064b8a71-af67-4daa-b07e-e8da79ffe28d", "File ID to download")
	// flag.Parse()

	// if *fileID == "" {
	// 	fmt.Println("Please provide --id=<file_id>")
	// 	return
	// }

	// conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	// if err != nil {
	// 	panic(err)
	// }
	// defer conn.Close()

	// client := pb.NewFileUploadServiceClient(conn)
	// stream, err := client.DownloadFile(context.Background(), &pb.DownloadRequest{FileId: *fileID})
	// if err != nil {
	// 	panic(err)
	// }

	// outFile, err := os.Create(fmt.Sprintf("downloaded_%s.zip", *fileID))
	// if err != nil {
	// 	panic(err)
	// }
	// defer outFile.Close()

	// for {
	// 	chunk, err := stream.Recv()
	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	outFile.Write(chunk.Content)
	// }

	// fmt.Println("✅ File downloaded successfully!")
	// time.Sleep(1 * time.Second)
}
