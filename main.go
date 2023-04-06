package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"

	"github.com/webrtc-demo-websocket/internal/signal"
)

var (
	wsMessageChannel         = make(chan []byte)
	wsLocalSessionDesChannel = make(chan []byte)
)

func main() {
	http.HandleFunc("/websocket", handleWebSocket)
	go http.ListenAndServe(":8081", nil)
	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	for msg := range wsMessageChannel {
		fmt.Println("message", string(msg))
		signal.Decode(string(msg), &offer)
		// if err := json.Unmarshal(msg, &offer); err != nil {
		// 	panic(err)
		// }

		// Create a new RTCPeerConnection
		peerConnection, err := webrtc.NewPeerConnection(config)
		if err != nil {
			panic(err)
		}
		defer func() {
			if cErr := peerConnection.Close(); cErr != nil {
				fmt.Printf("cannot close peerConnection: %v\n", cErr)
			}
		}()

		// Set the handler for Peer connection state
		// This will notify you when the peer has connected/disconnected
		peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
			fmt.Printf("Peer Connection State has changed: %s\n", s.String())
			if s.String() == "disconnected" {
				if cErr := peerConnection.Close(); cErr != nil {
					fmt.Printf("cannot close peerConnection: %v\n", cErr)
				}
			}
			if s == webrtc.PeerConnectionStateFailed {
				// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
				// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
				// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
				fmt.Println("Peer Connection has gone to failed exiting")
				os.Exit(0)
			}
		})

		// Register data channel creation handling
		peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

			// Register channel opening handling
			d.OnOpen(func() {
				// fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())
				startTime := time.Now()
				count := 0
				for range time.NewTicker(5 * time.Microsecond).C {
					message := signal.RandSeq(250)
					// fmt.Printf("Sending '%s'\n", message)
					if count%1000 == 0 {
						fmt.Printf("produce %d messages at speed %.2f/s\n", count, float64(count)/time.Since(startTime).Seconds())
					}
					// Send the message as text
					sendErr := d.SendText(message)
					if sendErr != nil {
						log.Println(sendErr)
						// panic(sendErr)
						return
					}
					count++
				}
			})

			// Register text message handling
			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
			})
		})

		// Set the remote SessionDescription
		err = peerConnection.SetRemoteDescription(offer)
		if err != nil {
			panic(err)
		}

		// Create an answer
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}

		// Sets the LocalDescription, and starts our UDP listeners
		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			panic(err)
		}

		// Output the answer in base64 so we can paste it in browser
		wsLocalSessionDesChannel <- []byte(signal.Encode(*peerConnection.LocalDescription()))
		fmt.Println(signal.Encode(*peerConnection.LocalDescription()))
	}

	go func() {
		close(wsMessageChannel)
	}()

	// Block forever
	select {}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		// Receive message from the WebSocket client
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		wsMessageChannel <- p
		err = conn.WriteMessage(messageType, <-wsLocalSessionDesChannel)
		if err != nil {
			log.Println(err)
			break
		}
	}
}
