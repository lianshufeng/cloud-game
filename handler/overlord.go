package handler

import (
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
)

// OverlordClient maintans connection to overlord
// We expect only one OverlordClient for each server
type OverlordClient struct {
	*cws.Client
}

// NewOverlordClient returns a client connecting to overlord for coordiation between different server
func NewOverlordClient(oc *websocket.Conn) *OverlordClient {
	if oc == nil {
		return nil
	}

	oclient := &OverlordClient{
		Client: cws.NewClient(oc),
	}
	return oclient
}

// RouteOverlord are all routes server received from overlord
func (h *Handler) RouteOverlord() {
	iceCandidates := [][]byte{}
	oclient := h.oClient

	// Received from overlord the serverID
	oclient.Receive(
		"serverID",
		func(response cws.WSPacket) (request cws.WSPacket) {
			// Stick session with serverID got from overlord
			log.Println("Received serverID ", response.Data)
			h.serverID = response.Data

			return cws.EmptyPacket
		},
	)

	oclient.Receive(
		"initwebrtc",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received relay SDP of a browser from overlord")
			peerconnection := webrtc.NewWebRTC()
			localSession, err := peerconnection.StartClient(resp.Data, iceCandidates, config.Width, config.Height)
			h.peerconnections[resp.SessionID] = peerconnection

			log.Println("Start peerconnection")
			if err != nil {
				if err != nil {
					log.Println("Error: Cannot create new webrtc session", err)
					return cws.EmptyPacket
				}
			}

			return cws.WSPacket{
				ID:        "sdp",
				Data:      localSession,
				SessionID: resp.SessionID,
			}
		},
	)

	// Received start from overlord. This happens when bridging
	// TODO: refactor
	//oclient.Receive(
	//"start",
	//func(resp cws.WSPacket) (req cws.WSPacket) {
	//log.Println("Received a start request from overlord")
	//log.Println("Add the connection to current room on the host ", resp.SessionID)

	//peerconnection := oclient.peerconnections[resp.SessionID]
	//log.Println("start session")

	////room := s.handler.createNewRoom(s.GameName, s.RoomID, s.PlayerIndex)
	//// Request room from Server if roomID is existed on the server
	//room := s.handler.getRoom(s.RoomID)
	//if room == nil {
	//log.Println("Room not found ", s.RoomID)
	//return cws.EmptyPacket
	//}
	//s.handler.detachPeerConn(s.peerconnection)
	//room.AddConnectionToRoom(peerconnection, s.PlayerIndex)
	////roomID, isNewRoom := startSession(peerconnection, resp.Data, resp.RoomID, resp.PlayerIndex)
	//log.Println("Done, sending back")

	//req.ID = "start"
	//req.RoomID = room.ID
	//return req
	//},
	//)

	oclient.Receive(
		"start",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a start request from overlord")
			log.Println("Add the connection to current room on the host ", resp.SessionID)
			peerconnection := h.peerconnections[resp.SessionID]
			roomID := h.startGameHandler(resp.Data, resp.RoomID, resp.PlayerIndex, peerconnection)
			return cws.WSPacket{
				ID:     "start",
				RoomID: roomID,
			}
		},
	)
	// heartbeat to keep pinging overlord. We not ping from server to browser, so we don't call heartbeat in browserClient
}

func getServerIDOfRoom(oc *OverlordClient, roomID string) string {
	log.Println("Request overlord roomID ", roomID)
	packet := oc.SyncSend(
		cws.WSPacket{
			ID:   "getRoom",
			Data: roomID,
		},
	)
	log.Println("Received roomID from overlord ", packet.Data)

	return packet.Data
}

func (h *Handler) startGameHandler(gameName, roomID string, playerIndex int, peerconnection *webrtc.WebRTC) string {
	//s.GameName = gameName
	//s.RoomID = roomID
	//s.PlayerIndex = playerIndex

	log.Println("Starting game")
	// If we are connecting to overlord, request corresponding serverID based on roomID
	// TODO: check if roomID is in the current server
	room := h.getRoom(roomID)
	log.Println("Got Room from local ", room, " ID: ", roomID)
	// If room is not running
	if room == nil {
		// Create new room
		room = h.createNewRoom(gameName, roomID, playerIndex)
		// Wait for done signal from room
		go func() {
			<-room.Done
			h.detachRoom(room.ID)
		}()
	}

	// Attach peerconnection to room. If PC is already in room, don't detach
	log.Println("Is PC in room", room.IsPCInRoom(peerconnection))
	if !room.IsPCInRoom(peerconnection) {
		h.detachPeerConn(peerconnection)
		room.AddConnectionToRoom(peerconnection, playerIndex)
	}

	// Register room to overlord if we are connecting to overlord
	if room != nil && h.oClient != nil {
		h.oClient.Send(cws.WSPacket{
			ID:   "registerRoom",
			Data: roomID,
		}, nil)
	}

	return room.ID
}

//func (s *Session) bridgeConnection(serverID string, gameName string, roomID string, playerIndex int) {
//log.Println("Bridging connection to other Host ", serverID)
//client := s.BrowserClient
//// Ask client to init

//log.Println("Requesting offer to browser", serverID)
//resp := client.SyncSend(cws.WSPacket{
//ID:   "requestOffer",
//Data: "",
//})

//// Ask overlord to relay SDP packet to serverID
//resp.TargetHostID = serverID
//log.Println("Sending offer to overlord to relay message to target host", resp.TargetHostID, "with payload")
//remoteTargetSDP := s.OverlordClient.SyncSend(resp)
//log.Println("Got back remote host SDP, sending to browser")
//// Send back remote SDP of remote server to browser
//s.BrowserClient.Send(cws.WSPacket{
//ID:   "sdp",
//Data: remoteTargetSDP.Data,
//}, nil)
//log.Println("Init session done, start game on target host")

//s.OverlordClient.SyncSend(cws.WSPacket{
//ID:           "start",
//Data:         gameName,
//TargetHostID: serverID,
//RoomID:       roomID,
//PlayerIndex:  playerIndex,
//})
//log.Println("Game is started on remote host")
//}