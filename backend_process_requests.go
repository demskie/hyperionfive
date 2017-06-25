package main

// 	bufferedRequests  []incomingRequest
// 	confirmedRequests []incomingRequest
// 	immediateRequests []incomingRequest
// 	workQueue         chan incomingRequest
//
// 	type incomingRequest struct {
// 		ClientID      string
// 		ClientMessage string
// 		ClientValues  []string
// 		ServerMessage string
// 	}

//localLock := sync.Mutex{}
//for coreNum := 0; coreNum < GLOBAL.numberOfCores; coreNum++ {
//	go func() {
//		for {
//			requests := []incomingRequest{}
//			singleRequest := <-GLOBAL.requestQueue
//			requests = append(requests, singleRequest)
//			for i := 0; i < 10000; i++ {
//				select {
//				case anotherRequest, doesExist := <-GLOBAL.requestQueue:
//					if doesExist {
//						requests = append(requests, anotherRequest)
//					}
//				default:
//					break
//				}
//			}
//			bufferClientRequests(localLock, requests)
//		}
//	}()
//}
