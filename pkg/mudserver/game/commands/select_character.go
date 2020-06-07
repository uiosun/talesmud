package commands

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/talesmud/talesmud/pkg/entities"
	"github.com/talesmud/talesmud/pkg/entities/characters"
	"github.com/talesmud/talesmud/pkg/entities/rooms"
	"github.com/talesmud/talesmud/pkg/mudserver/game/def"
	"github.com/talesmud/talesmud/pkg/mudserver/game/messages"
	m "github.com/talesmud/talesmud/pkg/mudserver/game/messages"
)

// SelectCharacterCommand ... select a character
type SelectCharacterCommand struct {
}

// Execute ... executes scream command
func (selectCharacter *SelectCharacterCommand) Execute(game def.GameCtrl, message *messages.Message) bool {

	parts := strings.Fields(message.Data)
	characterName := strings.Join(parts[1:], " ")

	if characters, err := game.GetFacade().CharactersService().FindByName(characterName); err == nil {

		for _, character := range characters {
			if character.Name == characterName && character.BelongsUserID == message.FromUser.ID.Hex() {
				// found character to select
				handleCharacterSelected(game, message.FromUser, character)
				return true
			}
		}
	}
	game.SendMessage(messages.Reply(message.FromUser.ID.Hex(), "Could not select character: "+characterName))

	return false
}

func handleCharacterSelected(game def.GameCtrl, user *entities.User, character *characters.Character) {

	// update player
	user.LastCharacter = character.ID.Hex()
	game.GetFacade().UsersService().Update(user.RefID, user)

	characterSelected := &messages.CharacterSelected{
		MessageResponse: messages.MessageResponse{
			Audience:   messages.MessageAudienceOrigin,
			AudienceID: user.ID.Hex(),
			Type:       messages.MessageTypeCharacterSelected,
			Message:    fmt.Sprintf("You are now playing as [%v]", character.Name),
		},
		Character: character,
	}

	game.SendMessage(characterSelected)

	var currentRoom *rooms.Room
	var err error
	// new character or not part of a room?
	if character.CurrentRoomID == "" {
		// find a random room to start in or get starting room
		rooms, _ := game.GetFacade().RoomsService().FindAll()

		if len(rooms) > 0 {
			// TOOD make this random or select a starting room
			currentRoom = rooms[0]

			//TODO: send this as message
			character.CurrentRoomID = currentRoom.ID.Hex()
			game.GetFacade().CharactersService().Update(character.ID.Hex(), character)

		}
	} else {
		if currentRoom, err = game.GetFacade().RoomsService().FindByID(character.CurrentRoomID); err != nil {
			log.WithField("room", character.CurrentRoomID).Warn("CurrentRoomID for player not found")
		}
	}

	// update room // send these state change messages via channel
	currentRoom.AddCharacter(character.ID.Hex())
	game.GetFacade().RoomsService().Update(currentRoom.ID.Hex(), currentRoom)

	enterRoom := m.NewEnterRoomMessage(currentRoom)
	enterRoom.AudienceID = user.ID.Hex()
	game.SendMessage(enterRoom)

	game.SendMessage(messages.CharacterJoinedRoom{
		MessageResponse: messages.MessageResponse{
			Audience:   m.MessageAudienceRoomWithoutOrigin,
			AudienceID: currentRoom.ID.Hex(),
			OriginID:   character.ID.Hex(),
			Message:    character.Name + " entered.",
		},
	})

}