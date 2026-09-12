// Package stations holds the built-in broadcast catalogue and the loader for
// user-supplied custom stations.
package stations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"iradio/internal/config"
)

// Station represents a broadcast channel.
type Station struct {
	ID        string `json:"id"`
	Region    string `json:"region"`
	NameZh    string `json:"name_zh"`
	NameEn    string `json:"name_en"`
	Dial      string `json:"dial"`
	Desc      string `json:"desc"`
	StreamURL string `json:"stream_url"`
}

// All is the full catalogue of built-in plus custom stations.
var All = []Station{
	// --- Hong Kong (RTHK) ---
	{
		ID:        "R1",
		Region:    "HK",
		NameZh:    "香港電台第一台",
		NameEn:    "RTHK Radio 1",
		Dial:      "FM 92.6 - 94.4 MHz",
		Desc:      "News, Current Affairs & Public Services (Cantonese)",
		StreamURL: "https://rthkaudio1-lh.akamaihd.net/i/radio1_1@355864/master.m3u8",
	},
	{
		ID:        "R2",
		Region:    "HK",
		NameZh:    "香港電台第二台",
		NameEn:    "RTHK Radio 2",
		Dial:      "FM 94.8 - 96.9 MHz",
		Desc:      "Youth, Pop Music & Community Promotion (Cantonese)",
		StreamURL: "https://rthkaudio2-lh.akamaihd.net/i/radio2_1@355865/master.m3u8",
	},
	{
		ID:        "R3",
		Region:    "HK",
		NameZh:    "香港電台第三台",
		NameEn:    "RTHK Radio 3",
		Dial:      "AM 567 / AM 1584 kHz / FM 97.9 - 106.8 MHz",
		Desc:      "News, Lifestyle & Music (English)",
		StreamURL: "https://rthkaudio3-lh.akamaihd.net/i/radio3_1@355866/master.m3u8",
	},
	{
		ID:        "R4",
		Region:    "HK",
		NameZh:    "香港電台第四台",
		NameEn:    "RTHK Radio 4",
		Dial:      "FM 97.6 - 98.9 MHz",
		Desc:      "Fine Music & Classical Arts (Bilingual)",
		StreamURL: "https://rthkaudio4-lh.akamaihd.net/i/radio4_1@355867/master.m3u8",
	},
	{
		ID:        "R5",
		Region:    "HK",
		NameZh:    "香港電台第五台",
		NameEn:    "RTHK Radio 5",
		Dial:      "AM 783 kHz / FM 92.3 - 106.8 MHz",
		Desc:      "Cultural Heritage, Elderly & Chinese Opera (Cantonese)",
		StreamURL: "https://rthkaudio5-lh.akamaihd.net/i/radio5_1@355868/master.m3u8",
	},
	{
		ID:        "PTH",
		Region:    "HK",
		NameZh:    "香港電台普通話台",
		NameEn:    "RTHK Putonghua Channel",
		Dial:      "AM 621 kHz / FM 100.9 - 103.3 MHz",
		Desc:      "News, Finance & Chinese Culture (Mandarin)",
		StreamURL: "https://rthkaudiopth-lh.akamaihd.net/i/radiopth_1@355869/master.m3u8",
	},
	{
		ID:        "R6",
		Region:    "HK",
		NameZh:    "香港電台第六台 (CNR)",
		NameEn:    "RTHK Radio 6 / Voice of HK",
		Dial:      "AM 675 kHz",
		Desc:      "Relay of China National Radio Voice of Hong Kong",
		StreamURL: "https://rthkaudio6cnr-lh.akamaihd.net/i/radio6cnr_1@575604/master.m3u8",
	},

	// --- Taiwan (Hit FM, News98, UFO, ICRT, Bravo, Classical) ---
	{
		ID:        "HITFM",
		Region:    "TW",
		NameZh:    "Hit FM 聯播網",
		NameEn:    "Hit FM 107.7",
		Dial:      "FM 107.7 MHz (Taipei)",
		Desc:      "Taiwan's #1 Hit Music Station (Taipei/Hitoradio)",
		StreamURL: "https://www.hitoradio.com/newweb/hichannel.php?channelID=1&action=getLIVEURL",
	},
	{
		ID:        "N98",
		Region:    "TW",
		NameZh:    "九八新聞台",
		NameEn:    "News98 FM 98.1",
		Dial:      "FM 98.1 MHz",
		Desc:      "Taiwan 24/7 News, Finance, Analysis & Talk (Taipei)",
		StreamURL: "https://stream.rcs.revma.com/pntx1639ntzuv.m4a",
	},
	{
		ID:        "UFO",
		Region:    "TW",
		NameZh:    "飛碟聯播網",
		NameEn:    "UFO Radio FM 92.1",
		Dial:      "FM 92.1 MHz",
		Desc:      "Pop Music, Talk Shows & Lifestyle (Taiwan)",
		StreamURL: "https://stream.rcs.revma.com/em90w4aeewzuv.m4a",
	},
	{
		ID:        "ICRT",
		Region:    "TW",
		NameZh:    "台北國際社區廣播電台",
		NameEn:    "ICRT FM 100",
		Dial:      "FM 100.7 MHz",
		Desc:      "Taiwan's Premier English Radio Station",
		StreamURL: "https://stream.rcs.revma.com/nkdfurztxp3vv",
	},
	{
		ID:        "BRAVO",
		Region:    "TW",
		NameZh:    "台北都會音樂台",
		NameEn:    "Bravo FM 91.3",
		Dial:      "FM 91.3 MHz",
		Desc:      "Jazz, Classical & Urban Lifestyle (Taipei)",
		StreamURL: "https://onair.bravo913.com.tw:9130/live.mp3",
	},
	{
		ID:        "CFM",
		Region:    "TW",
		NameZh:    "好家庭古典音樂台",
		NameEn:    "Classical FM 97.7",
		Dial:      "FM 97.7 MHz",
		Desc:      "Classical Music, Philosophy & Arts (Taichung)",
		StreamURL: "https://onair.family977.com.tw:8977/live.mp3",
	},
	{
		ID:        "BCCN",
		Region:    "TW",
		NameZh:    "中廣新聞網",
		NameEn:    "BCC News Radio",
		Dial:      "AM 648 kHz / App",
		Desc:      "Taiwan BCC 24/7 Professional News Network",
		StreamURL: "https://stream.rcs.revma.com/78fm9wyy2tzuv",
	},
	{
		ID:        "BCCM",
		Region:    "TW",
		NameZh:    "中廣音樂網 (i radio)",
		NameEn:    "BCC i Radio / Music",
		Dial:      "FM 96.3 MHz / Web",
		Desc:      "Taiwan & Mandopop Non-stop Music Station",
		StreamURL: "https://stream.rcs.revma.com/ndk05tyy2tzuv",
	},
	{
		ID:        "BCCP",
		Region:    "TW",
		NameZh:    "中廣流行網",
		NameEn:    "BCC i like radio",
		Dial:      "FM 103.3 MHz",
		Desc:      "Taiwan Pop Culture, Lifestyle & Talk",
		StreamURL: "https://stream.rcs.revma.com/aw9uqyxy2tzuv",
	},

	// --- Japan (Shonan Beach, OTTAVA, FM Setagaya, AnimeNfo) ---
	{
		ID:        "SBFM",
		Region:    "JP",
		NameZh:    "湘南ビーチFM",
		NameEn:    "Shonan Beach FM",
		Dial:      "FM 78.9 MHz (Kanagawa)",
		Desc:      "Jazz, City Pop, Oldies & Coastal Vibes (Hayama/Zushi)",
		StreamURL: "https://shonanbeachfm.out.airtime.pro:8000/shonanbeachfm_a",
	},
	{
		ID:        "OTV",
		Region:    "JP",
		NameZh:    "OTTAVA 經典音樂台",
		NameEn:    "OTTAVA Classic Radio",
		Dial:      "Online (Tokyo)",
		Desc:      "Japan's Premier Classical Music & Arts Broadcast",
		StreamURL: "https://ottava2.out.airtime.pro/ottava2_a",
	},
	{
		ID:        "FMS834",
		Region:    "JP",
		NameZh:    "エフエム世田谷",
		NameEn:    "FM Setagaya 83.4",
		Dial:      "FM 83.4 MHz (Tokyo)",
		Desc:      "Tokyo Community Broadcast, News, Culture & Pop",
		StreamURL: "https://fmsetagaya834.out.airtime.pro/fmsetagaya834_a",
	},
	{
		ID:        "ANIFO",
		Region:    "JP",
		NameZh:    "AnimeNfo 動畫音樂台",
		NameEn:    "AnimeNfo Radio",
		Dial:      "Online / Tokyo",
		Desc:      "24/7 Anime OSTs, J-Pop, Vocaloid & Game Music",
		StreamURL: "https://stream.animenfo.com:8000/live",
	},

	// --- Singapore (YES 933, CNA938, Class 95, UFM 100.3, Kiss92) ---
	{
		ID:        "YES933",
		Region:    "SG",
		NameZh:    "YES 933 頂尖流行音樂台",
		NameEn:    "YES 933 FM",
		Dial:      "FM 93.3 MHz (Singapore)",
		Desc:      "Singapore's #1 Mandarin Hit Music Station (Mediacorp)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/YES933AAC.aac",
	},
	{
		ID:        "CNA938",
		Region:    "SG",
		NameZh:    "CNA938 新加坡新聞台",
		NameEn:    "CNA938 News Radio",
		Dial:      "FM 93.8 MHz (Singapore)",
		Desc:      "Singapore 24/7 News, Business & Current Affairs",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/938NOWAAC.aac",
	},
	{
		ID:        "CLS95",
		Region:    "SG",
		NameZh:    "Class 95 英語音樂台",
		NameEn:    "Class 95 FM",
		Dial:      "FM 95.0 MHz (Singapore)",
		Desc:      "The Best Mix of Music & Adult Contemporary",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/CLASS95AAC.aac",
	},
	{
		ID:        "UFM100",
		Region:    "SG",
		NameZh:    "UFM 100.3 流行音樂",
		NameEn:    "UFM 100.3 FM",
		Dial:      "FM 100.3 MHz (Singapore)",
		Desc:      "Top Mandopop Hits & Dynamic Morning Talk (SPH)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/UFM_1003AAC.aac",
	},
	{
		ID:        "KISS92",
		Region:    "SG",
		NameZh:    "Kiss92 英語流行台",
		NameEn:    "Kiss92 FM",
		Dial:      "FM 92.0 MHz (Singapore)",
		Desc:      "All The Great Songs In One Place (SPH)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/KISS_92AAC.aac",
	},

	// --- Malaysia (988, Ai FM, BFM 89.9, Suria FM) ---
	{
		ID:        "M988",
		Region:    "MY",
		NameZh:    "988電台",
		NameEn:    "988 FM",
		Dial:      "FM 98.8 MHz (Kuala Lumpur)",
		Desc:      "Malaysia Top Chinese Hit Music & News Network",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/988_FMAAC.aac",
	},
	{
		ID:        "AIFM",
		Region:    "MY",
		NameZh:    "爱FM (Ai FM)",
		NameEn:    "Ai FM Malaysia",
		Dial:      "FM 89.3 / 106.7 MHz",
		Desc:      "RTM National Chinese Radio Station (Mandarin)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/AI_FMAAC.aac",
	},
	{
		ID:        "BFM899",
		Region:    "MY",
		NameZh:    "BFM 商業電台",
		NameEn:    "BFM 89.9",
		Dial:      "FM 89.9 MHz (Kuala Lumpur)",
		Desc:      "The Business Station, Current Affairs & Finance (English)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/BFM.mp3",
	},
	{
		ID:        "SURIA",
		Region:    "MY",
		NameZh:    "Suria FM 陽光電台",
		NameEn:    "Suria FM 105.3",
		Dial:      "FM 105.3 MHz (Klang Valley)",
		Desc:      "Segar & Terkini, Malay Hits & Entertainment (Malay)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/SURIA_FMAAC.aac",
	},
}

// LoadCustom reads user-provided stations from the config directory.
func LoadCustom() []Station {
	dir := config.Dir()
	candidates := []string{
		filepath.Join(dir, "stations.json"),
		filepath.Join(dir, "custom_stations.json"),
	}

	var loaded []Station
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var list []Station
		if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
			loaded = append(loaded, list...)
		}
	}
	return loaded
}

// InitCustom merges custom stations into All and returns how many were loaded.
func InitCustom() int {
	custom := LoadCustom()
	if len(custom) == 0 {
		return 0
	}

	idxMap := make(map[string]int)
	for i, s := range All {
		idxMap[s.ID] = i
	}

	for i, s := range custom {
		if s.StreamURL == "" {
			continue
		}
		if s.ID == "" {
			s.ID = fmt.Sprintf("CUST%d", i+1)
		}
		if s.Region == "" {
			s.Region = "USER"
		}
		if s.NameZh == "" && s.NameEn != "" {
			s.NameZh = s.NameEn
		} else if s.NameEn == "" && s.NameZh != "" {
			s.NameEn = s.NameZh
		}

		if idx, exists := idxMap[s.ID]; exists {
			All[idx] = s
		} else {
			All = append(All, s)
			idxMap[s.ID] = len(All) - 1
		}
	}
	return len(custom)
}