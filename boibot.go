package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/line/line-bot-sdk-go/linebot"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var bot *linebot.Client
func callbackHandler(c *gin.Context) {
	var err error
	var CHANNEL_SECRET = viper.GetString("boibot.channelSecret")
	var CHANNEL_TOKEN = viper.GetString("boibot.channelToken")
	if err = viper.ReadInConfig();
		err != nil {
		log.Errorln("Fatal Error Config File: ",err)
		panic("Fatal Error Config File")
	}
	//connect to line_bot
	bot, err = linebot.New(
		CHANNEL_SECRET,
		CHANNEL_TOKEN,
	)
	if err != nil {
		log.Fatal(err)
	}
	events, err := bot.ParseRequest(c.Request)//message

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			c.JSON(400, gin.H{}) //Bad Request
		} else {
			c.JSON(500, gin.H{}) //Internet Server Error
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				log.Println("===== " + message.Text + " =====")
				text := strings.Split(strings.ToLower(message.Text), " ")
				if strings.Trim(text[0], " ") != "boibot" {
					return
				}
				boibotCmd := text[1]

				if reply(event.ReplyToken, message.Text) {
					return
				}
				if 	boibotCmd == "โควิด" {
					reportCovidTH(event.ReplyToken, text)
					return
				}
				if 	boibotCmd == "covid" {
					reportCovidEN(event.ReplyToken, text)
					return
				}
				if 	boibotCmd == "top3" {
					reportCovidTop3(event.ReplyToken, text)
					return
				}
				if boibotCmd == "help" {
					text := fmt.Sprintf("คำสั่งเบื้องต้น\nboibot (โควิด, covid) (รหัสประเทศ)\n" +
						"=> เเสดงข้อมูลเกี่ยวกับ covid19 ของประเทศต่างๆ \nเช่น \"boibot covid th\"\n\n" +
						"boibot top3\n=> เป็นการเเสดง 3 อันดับประเทศที่มียอดผู้ติดเชื้อมากที่สุดเเละเเสดงอันดับของประเทศไทยด้วย")
					sendReplyMessage(event.ReplyToken, text)
					return
				}

				if boibotCmd == "get out" || boibotCmd == "ออกไป" {
					sendReplyMessage(event.ReplyToken, "บ๊ายบาย")

					log.Println("event.Source.Type:", event.Source.Type)
					log.Println("event.Source.GroupID:", event.Source.GroupID)
					log.Println("event.Source.RoomID:", event.Source.RoomID)

					leaveUrl := fmt.Sprintf("https://api.line.me/v2/bot/group/%s/leave", event.Source.GroupID)
					if event.Source.Type == "room" {
						leaveUrl = fmt.Sprintf("https://api.line.me/v2/bot/room/%s/leave", event.Source.RoomID)
					}
					post, err := http.NewRequest("POST", leaveUrl, nil)
					post.Header.Set("Authorization", "Bearer "+ CHANNEL_TOKEN)
					client := &http.Client{
						Timeout: 10 * time.Second,
					}
					apiRes, err := client.Do(post)
					if err != nil {
						log.Println("Cannot post API leave group:", err)
					}
					defer apiRes.Body.Close()
				}

			case *linebot.StickerMessage:
				log.Println("StickerMessage ================")
				log.Println("event.Source.UserID:", event.Source.UserID)
			case *linebot.LocationMessage:
				log.Println("LocationMessage ================")
			case *linebot.ImageMessage:
				log.Println("ImageMessage ================")
			default:
				//sendReplyMessage(event.ReplyToken, "Sorry, this command is not support.")
			}
		}
	}
}

func reportCovidTH(replyToken string, text []string) bool {
	code := strings.ToUpper(text[2])
	country ,err:= getCountryByCode(code)
	if err != nil {
		sendReplyMessage(replyToken, err.Error())
	}
	data ,err:= getTotalPatientsByCountryId(country.Id)
	if err != nil {
		sendReplyMessage(replyToken, err.Error())
	}
	Credit := "Credit: https://thevirustracker.com/"
	if code == "TH" {
		reportData, err:= getReportThailand()
		if err != nil {
			sendReplyMessage(replyToken, err.Error())
		}
		PercentCaseIncrease :=  (float32(data.TotalCasesIncreases)  / float32 (reportData[len(reportData)-2].Confirmed)) * 100
		PercentDeathIncrease := (float32(data.TotalDeathsIncreases)  / float32 (reportData[len(reportData)-2].Deaths)) * 100
		Credit = "Credit: https://covid19.th-stat.com/th"
		message := fmt.Sprintf("ยืนยันผู้ติดเชื้อ Covid19  \U0001f9a0\n" +
			"ทั้งหมดใน%s  %s  \n" +
			"เมื่อ %s \n" +
			"ผู้ติดเชื้อ %d คน[+%d](+%.2f%s)  \U0001F637\n" +
			"กำลังรักษา %d คน  \U0001F3E5\n" +
			"หายเเล้ว %d คน  \U0001F606\n" +
			"เสียชีวิต %d คน[+%d](+%.2f%s)  \U0001F480\n" +
			"%s",
			country.CountryTh, country.Emoji ,data.UpdateDate[0:10], data.TotalCases, data.TotalCasesIncreases,
			PercentCaseIncrease, "%", data.TotalActiveCases, data.TotalRecovered, data.TotalDeaths, data.TotalDeathsIncreases,
			PercentDeathIncrease, "%", Credit)
		sendReplyMessage(replyToken, message)
	}else {
		message := fmt.Sprintf("ยืนยันผู้ติดเชื้อ Covid19  \U0001f9a0\n" +
			"ทั้งหมดใน%s  %s  \n" +
			"เมื่อ %s \n" +
			"ผู้ติดเชื้อ %d คน[+%d]\U0001F637\n" +
			"กำลังรักษา %d คน  \U0001F3E5\n" +
			"หายเเล้ว %d คน  \U0001F606\n" +
			"เสียชีวิต %d คน[+%d]	\U0001F480\n" +
			"%s",
			country.CountryTh, country.Emoji ,data.UpdateDate[0:10],
			data.TotalCases,data.TotalCasesIncreases,
			data.TotalActiveCases, data.TotalRecovered,
			data.TotalDeaths,data.TotalDeathsIncreases,
			Credit)
		sendReplyMessage(replyToken, message)
	}
	return true
}

func reportCovidEN(replyToken string, text []string) bool {
	code := strings.ToUpper(text[2])
	country, err:= getCountryByCode(code)
	if err != nil {
		sendReplyMessage(replyToken, err.Error())
	}
	data ,err:= getTotalPatientsByCountryId(country.Id)
	if err != nil {
		sendReplyMessage(replyToken, err.Error())
	}
	Credit := "Credit: https://thevirustracker.com/"
	if code == "TH" {
		reportData, err:= getReportThailand()
		if err != nil {
			sendReplyMessage(replyToken, err.Error())
		}
		PercentCaseIncrease :=  (float32(data.TotalCasesIncreases)  / float32 (reportData[len(reportData)-2].Confirmed)) * 100
		PercentDeathIncrease := (float32(data.TotalDeathsIncreases)  / float32 (reportData[len(reportData)-2].Deaths)) * 100
		Credit = "Credit: https://covid19.th-stat.com/th"
		message := fmt.Sprintf("Covid19  \U0001f9a0  in Thailand  %s  \n" +
			"UpdateDate %s \n" +
			"Confirmed %d [+%d](+%.2f%s)  \U0001F637\n" +
			"Hospitalized %d   \U0001F3E5\n" +
			"Recovered %d  \U0001F606\n" +
			"Deaths %d [+%d](+%.2f%s)  \U0001F480\n" +
			"%s",
			country.Emoji ,data.UpdateDate[0:10], data.TotalCases, data.TotalCasesIncreases,
			PercentCaseIncrease, "%", data.TotalActiveCases, data.TotalRecovered, data.TotalDeaths, data.TotalDeathsIncreases,
			PercentDeathIncrease, "%", Credit)
		sendReplyMessage(replyToken, message)
	}else {
		message := fmt.Sprintf("Covid19  \U0001f9a0  in %s  %s  \n" +
			"UpdateDate %s \n" +
			"Confirmed %d [+%d]  \U0001F637\n" +
			"Hospitalized %d   \U0001F3E5\n" +
			"Recovered %d  \U0001F606\n" +
			"Deaths %d [+%d] \U0001F480\n" +
			"%s",
			country.CountryEng, country.Emoji ,data.UpdateDate[0:10],
			data.TotalCases, data.TotalCasesIncreases,
			data.TotalActiveCases, data.TotalRecovered,
			data.TotalDeaths, data.TotalDeathsIncreases,
			Credit)
		sendReplyMessage(replyToken, message)
	}
	return true
}

func reply(replyToken string, message string) bool {
	if message == "" {
		return false
	}

	if m, _ := regexp.MatchString("boibot ด่า.*?ให้หน่อย", message); m {
		name := strings.TrimSpace(message[len("boibot ด่า") : len(message)-len("ให้หน่อย")])

		dar := []string{"อีข้อศอกหมี", "อีตาปลาถูกตัดที่ร้านทำเล็บ", "อีกิ้งกือตัดต่อพันธุกรรม", "อีเล็บขบของไส้เดือน", "ไอ้แตงกวาดอง", "ไอ้กะหล่ำปลี", "อีเห็ดสด",
			"อีแมวน้ำ", "ไอ้ปูปู้", "อิอมีบาวิ่งผ่านพารามีเซียม", "อีปลวกมีปีก", "อีแบรนด์ซุปไก่สกัด", "อิโดเรม่อนไม่มีกระเป๋าวิเศษ", "อิกระดาษโดนน้ำ", "อีสายพานจักรยาน",
			"อีmouseไม่มีwheel", "อีCPU single core", "อีpower bank แบตหมด", "อีสาย usb หักใน", "อิหอยกาบสแกนดิเนเวีย", "อิต่อต้านอนุมูลอิสระ",
			"อีส้มตำไม่ใส่มะละกอ", "อี Ferrari ยกสูง", "อิน้ำยาปรับผ้านุ่ม", "อิดาบเจ็ดสี มณีเจ็ดแสง", "อีCPUริมๆWafer", "อีPower supply 200W",
			"อี Protoss ไม่มี carrier", "อีไข่เจียวไม่ใส่หมูสับ", "อี DNA เส้นตรง", "ไอ้ตุ๊กตาปูปู้", "ไอ้ผัดไทยห่อไข่ดาว", "ไอ้กระทู้พันทิป", "ไอ้แว่นตาเลนส์เว้า",
			"ไอ้หลอดไฟสี daylight", "ไอ้เสื้อยืดคอเต่า", "ไอ้เสื้อลายสก๊อต", "ไอ้หนังสือพิมพ์เปื้อนนิ้ว", "ไอ้นาฬิกา Kinetic", "ไอ้ Siri text mode",
			"ไอ้ดอกกุหลาบหนามแหลม", "อี Twitter limit 140 ตัวอักษร", "อีเบียร์ใส่น้ำแข็ง", "อีไวน์หมัก10ปี"}

		sendReplyMessage(replyToken, name+" "+dar[rand.Intn(len(dar))])
		return true
	}


	if strings.HasPrefix(message,"boibot /?") {
		sendReplyMessage(replyToken, "ไม่ช่วย ไม่ตอบ")
		return true
	}
	if strings.HasPrefix(message,"boibot แสด") {
		sendReplyMessage(replyToken, "ด่าตัวเองหรอ?")
		return true
	}
	if strings.HasPrefix(message, "boibot thank") {
		sendReplyMessage(replyToken, "เก็บคำนั้นไว้กับนายเถอะ")
		return true
	}
	if strings.HasPrefix(message, "boibot ใครหน้าหีที่สุดในกลุ่ม") {
		sendReplyMessage(replyToken, "ไอบิ๊ก")
		return true
	}
	if strings.HasPrefix(message, "boibot resurrect") {
		sendReplyMessage(replyToken, "ชั้นจะกลับมาเสมอ แม้นายจะเตะชั้นอีกกี่ครั้ง")
		return true
	}
	if message == "วันก่อนครับ" {
		sendReplyMessage(replyToken, "ทำไมหรอครับ?")
		return true
	}
	if strings.HasPrefix(message, "มีคุณยายขึ้นรถเมล์ แม่งไม่มีคนลุกเลยครับ") {
		sendReplyMessage(replyToken, "ไม่มีน้ำใจกันเลยนะครับ")
		return true
	}
	if strings.HasPrefix(message, "ซักพักมีผู้ชายคนนึงทนไม่ไหว ลุกให้ยายนั่ง คนร้องกันทั้งรถเลยครับ") {
		sendReplyMessage(replyToken, "เพราะชื่นชมที่เค้าเป็นสุภาพบุรุษ?")
		return true
	}
	if strings.HasPrefix(message, "เปล่า คนที่ลุกให้ยายนั่งอะ คนขับ") {
		sendReplyMessage(replyToken, "...")
		return true
	}
	if strings.HasPrefix(message, "ไปสวนสาธารณะเปิดใหม่มา") {
		sendReplyMessage(replyToken, "ไปเดินเล่นหรอครับ?")
		return true
	}
	if strings.HasPrefix(message, "ไปถึงนี่ ไม่มีที่ให้นั่งเลยครับ") {
		sendReplyMessage(replyToken, "คนเยอะมาก ใครๆก็ไป จนไม่มีที่นั่ง?")
		return true
	}
	if strings.HasPrefix(message, "เปล่า มีแต่ม้านั่งครับ...") {
		sendReplyMessage(replyToken, "แสดด")
		return true
	}
	if strings.HasPrefix(message, "boibot เก่งมาก") {
		sendReplyMessage(replyToken, "ไม่ต้องมาแกล้งชมชั้นหรอก")
		return true
	}
	if strings.HasPrefix(message, "เฮ้ย ชมจริงๆ") {
		sendReplyMessage(replyToken, "อ่ะๆ กองไว้ตรงนั้นแหละ")
		return true
	}
	if strings.HasPrefix(message, "boibot ขอบใจนะ") {
		sendReplyMessage(replyToken, "เก็บคำนั้นไว้เถอะ")
		return true
	}
	if strings.HasPrefix(message, "boibot เขียนโปรแกรมให้หน่อยได้มะ") {
		sendReplyMessage(replyToken, "วันก่อนครับ")
		return true
	}
	if strings.HasPrefix(message, "ทำไมหรอครับ??") {
		sendReplyMessage(replyToken, "มีฝรั่งดูโค้ดผม บอกว่าโค้ดผมสะอาดมากเลยครับ")
		return true
	}
	if strings.HasPrefix(message, "เค้าพูดว่าไรหรอครับ??") {
		sendReplyMessage(replyToken, "ยัวร์ โค้ด ซัก")
		return true
	}

	return false
}

func sendReplyMessage(replyToken string, message string) error {
	if _, err := bot.ReplyMessage(replyToken,
		linebot.NewTextMessage(message)).Do(); err != nil {
		log.Print(replyToken)
		log.Print(message)
		return err
	}
	return nil
}

//Hard code is OP
func reportCovidTop3(replyToken string, text []string) bool {
	data ,err:= GetTotalTop3()
	if err != nil {
		sendReplyMessage(replyToken, err.Error())
	}

	Credit := "Credit: https://thevirustracker.com/"

	message := fmt.Sprintf("Top3 ประเทศที่มียอดผู้ติดเชื้อมากที่สุด Covid19  \U0001f9a0\n\n" +
		"อันดับที่ %d  \n" +
		"ประเทศ %s (%s)\n" +
		"ผู้ติดเชื้อ %d คน[+%d]  \U0001F637\n" +
		"กำลังรักษา %d คน  \U0001F3E5\n" +
		"หายเเล้ว %d คน  \U0001F606\n" +
		"เสียชีวิต %d คน[+%d]  \U0001F480\n\n" +

		"อันดับที่ %d  \n" +
		"ประเทศ %s (%s)\n" +
		"ผู้ติดเชื้อ %d คน[+%d]  \U0001F637\n" +
		"กำลังรักษา %d คน  \U0001F3E5\n" +
		"หายเเล้ว %d คน  \U0001F606\n" +
		"เสียชีวิต %d คน[+%d]  \U0001F480\n\n" +

		"อันดับที่ %d  \n" +
		"ประเทศ %s (%s)\n" +
		"ผู้ติดเชื้อ %d คน[+%d]  \U0001F637\n" +
		"กำลังรักษา %d คน  \U0001F3E5\n" +
		"หายเเล้ว %d คน  \U0001F606\n" +
		"เสียชีวิต %d คน[+%d]  \U0001F480\n\n" +

		"อันดับที่ %d  \n" +
		"ประเทศ %s (%s)\n" +
		"ผู้ติดเชื้อ %d คน[+%d]  \U0001F637\n" +
		"กำลังรักษา %d คน  \U0001F3E5\n" +
		"หายเเล้ว %d คน  \U0001F606\n" +
		"เสียชีวิต %d คน[+%d]  \U0001F480\n\n\n" +

		"วันที่อัพเดทล่าสุด %s " +
		"\n%s",
		data[0].Sequence, data[0].CountryNameTH, data[0].CountryNameEN,data[0].TotalCases, data[0].TotalCasesIncreases,
		data[0].TotalActiveCases, data[0].TotalRecovered, data[0].TotalDeaths, data[0].TotalDeathsIncreases,

		data[1].Sequence, data[1].CountryNameTH, data[1].CountryNameEN,data[1].TotalCases, data[1].TotalCasesIncreases,
		data[1].TotalActiveCases, data[1].TotalRecovered, data[1].TotalDeaths, data[1].TotalDeathsIncreases,

		data[2].Sequence, data[2].CountryNameTH, data[2].CountryNameEN,data[2].TotalCases, data[2].TotalCasesIncreases,
		data[2].TotalActiveCases, data[2].TotalRecovered, data[2].TotalDeaths, data[2].TotalDeathsIncreases,

		data[3].Sequence, data[3].CountryNameTH, data[3].CountryNameEN,data[3].TotalCases, data[3].TotalCasesIncreases,
		data[3].TotalActiveCases, data[3].TotalRecovered, data[3].TotalDeaths, data[3].TotalDeathsIncreases,

		data[0].UpdateDate[0:10], Credit)
	sendReplyMessage(replyToken, message)
	return true
}

