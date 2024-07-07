package bot

import (
	"encoding/json"
	"fmt"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	"github.com/XiaoMengXinX/Music163Api-Go/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func processLyric(message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	var msgResult tgbotapi.Message
	sendFailed := func() {
		editMsg := tgbotapi.NewEditMessageText(msgResult.Chat.ID, msgResult.MessageID, fmt.Sprintf(getLrcFailed))
		_, err = bot.Send(editMsg)
		if err != nil {
			logrus.Errorln(err)
		}
	}
	if message.CommandArguments() == "" && message.ReplyToMessage == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, inputContent)
		msg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(msg)
		return err
	} else if message.CommandArguments() == "" && message.ReplyToMessage != nil {
		message = *message.ReplyToMessage
		if !message.IsCommand() && len(message.Entities) != 0 {
			message.Entities[0].Type = "bot_command"
			message.Entities[0].Length = -1
			message.Entities[0].Offset = 0
		} else if !message.IsCommand() && len(message.Entities) == 0 {
			message.Entities = []tgbotapi.MessageEntity{{Type: "bot_command", Length: -1, Offset: 0}}
		}
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, fetchingLyric)
	msg.ReplyToMessageID = message.MessageID
	msgResult, err = bot.Send(msg)
	if err != nil {
		return err
	}

	musicID := parseMusicID(message.CommandArguments())
	if musicID == 0 {
		searchResult, _ := api.SearchSong(data, api.SearchSongConfig{
			Keyword: message.CommandArguments(),
			Limit:   5,
		})
		if len(searchResult.Result.Songs) == 0 {
			editMsg := tgbotapi.NewEditMessageText(msgResult.Chat.ID, msgResult.MessageID, noResults)
			_, err = bot.Send(editMsg)
			return err
		}
		musicID = searchResult.Result.Songs[0].Id
	}

	b := api.NewBatch(api.BatchAPI{
		Key:  api.SongLyricAPI,
		Json: api.CreateSongLyricReqJson(musicID),
	}, api.BatchAPI{
		Key:  api.SongDetailAPI,
		Json: api.CreateSongDetailReqJson([]int{musicID}),
	}).Do(data)
	if b.Error != nil {
		sendFailed()
		return b.Error
	}

	_, result := b.Parse()
	var lyric types.SongLyricData
	var detail types.SongsDetailData
	_ = json.Unmarshal([]byte(result[api.SongLyricAPI]), &lyric)
	_ = json.Unmarshal([]byte(result[api.SongDetailAPI]), &detail)

	if lyric.Lrc.Lyric != "" && len(detail.Songs) != 0 {
		lyricText := "```lrc\n" + lyric.Lrc.Lyric + "\n```"
		newMsg := tgbotapi.NewMessage(message.Chat.ID, lyricText)
		newMsg.ParseMode = tgbotapi.ModeMarkdown
		newMsg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(newMsg)
		if err != nil {
			return err
		}
		deleteMsg := tgbotapi.NewDeleteMessage(msgResult.Chat.ID, msgResult.MessageID)
		_, err = bot.Request(deleteMsg)
		return err
	}
	sendFailed()
	return
}
