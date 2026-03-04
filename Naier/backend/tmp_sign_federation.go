package main

import (
  "crypto/ed25519"
  "crypto/rand"
  "encoding/base64"
  "encoding/json"
  "os"
  "time"
)

type User struct {
  ID string `json:"id"`
  Username string `json:"username"`
  DisplayName string `json:"display_name"`
  PublicKey string `json:"public_key"`
  IdentitySigningKey string `json:"identity_signing_key"`
  IdentityExchangeKey string `json:"identity_exchange_key"`
  AvatarURL string `json:"avatar_url"`
  Bio string `json:"bio"`
  ServerID string `json:"server_id"`
  CreatedAt time.Time `json:"created_at"`
}

type Member struct {
  User User `json:"user"`
  Role string `json:"role"`
  JoinedAt time.Time `json:"joined_at"`
  IsMuted bool `json:"is_muted"`
}

type Payload struct {
  ChannelID string `json:"channel_id"`
  ChannelType string `json:"channel_type"`
  Name string `json:"name"`
  Description string `json:"description"`
  IsEncrypted bool `json:"is_encrypted"`
  MaxMembers int `json:"max_members"`
  MemberCount int `json:"member_count"`
  Members []Member `json:"members"`
}

type Event struct {
  EventID string `json:"event_id"`
  Type string `json:"type"`
  ServerID string `json:"server_id"`
  Timestamp time.Time `json:"timestamp"`
  Payload json.RawMessage `json:"payload"`
  Signature string `json:"signature"`
}

type Signable struct {
  EventID string `json:"event_id"`
  Type string `json:"type"`
  ServerID string `json:"server_id"`
  Timestamp time.Time `json:"timestamp"`
  Payload json.RawMessage `json:"payload"`
}

type Envelope struct { Event Event `json:"event"` }

func main() {
  pub, priv, _ := ed25519.GenerateKey(rand.Reader)
  now := time.Now().UTC()
  payload := Payload{ChannelID:"remote-room-3", ChannelType:"group", Name:"Remote Ops 3", Description:"Shadow sync test 3", IsEncrypted:true, MaxMembers:20, MemberCount:1, Members: []Member{{User: User{ID:"remote-user-3", Username:"remotecarol", DisplayName:"Remote Carol", PublicKey:"remote-public-3", IdentitySigningKey:"remote-sign-3", IdentityExchangeKey:"remote-exchange-3", AvatarURL:"", Bio:"", ServerID:"remote3.test", CreatedAt:now}, Role:"owner", JoinedAt:now, IsMuted:false}}}
  payloadBytes, _ := json.Marshal(payload)
  evt := Event{EventID:"evt-shadow-go-3", Type:"CHANNEL_STATE_SYNC", ServerID:"remote3.test", Timestamp:now, Payload:payloadBytes}
  signableBytes, _ := json.Marshal(Signable{EventID:evt.EventID, Type:evt.Type, ServerID:evt.ServerID, Timestamp:evt.Timestamp.UTC(), Payload:evt.Payload})
  evt.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(priv, signableBytes))
  eventBytes, _ := json.Marshal(Envelope{Event:evt})
  metaBytes, _ := json.Marshal(map[string]string{"publicKey": base64.RawStdEncoding.EncodeToString(pub)})
  _ = os.WriteFile("tmp_federation_event.json", eventBytes, 0644)
  _ = os.WriteFile("tmp_federation_meta.json", metaBytes, 0644)
}
