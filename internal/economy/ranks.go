package economy

// RankInfo mendefinisikan informasi tingkatan pangkat pemain berdasarkan saldo chip.
type RankInfo struct {
	Name        string
	MinChips    int
	Emoji       string
	Styled      string
	Description string
}

// Ranks menampung daftar pangkat yang diurutkan dari yang tertinggi ke terendah.
// Penggunaan font unicode premium membuat tampilan pamer pangkat di WhatsApp sangat mencolok.
var Ranks = []RankInfo{
	{
		Name:        "Hegemon",
		MinChips:    10000000,
		Emoji:       "🌌",
		Styled:      "🌌 *𝕳𝖊𝖌𝖊𝖒𝖔𝖓* 🌌",
		Description: "Sosok puncak yang menetapkan standar dunia dan menjadi penggerak utama dalam sejarah peradaban.",
	},
	{
		Name:        "Emperor",
		MinChips:    5000000,
		Emoji:       "🏛️",
		Styled:      "🏛️ *𝙀𝙢𝙥𝙚𝙧𝙤𝙧*",
		Description: "Pemersatu agung yang menyelaraskan berbagai keberagaman di bawah naungan kepemimpinannya.",
	},
	{
		Name:        "Monarch",
		MinChips:    1000000,
		Emoji:       "👑",
		Styled:      "👑 *𝑴𝒐𝒏𝒂𝒓𝒄𝒉*",
		Description: "Penguasa berwibawa yang memegang teguh komitmen untuk stabilitas dan arah negerinya.",
	},
	{
		Name:        "Strategist",
		MinChips:    500000,
		Emoji:       "🔱",
		Styled:      "🔱 *𝕊𝕥𝕣𝕒𝕥𝕖𝕘𝕚𝕤𝕥*",
		Description: "Arsitek agung yang merancang langkah masa depan dengan perhitungan yang melampaui zaman.",
	},
	{
		Name:        "General",
		MinChips:    250000,
		Emoji:       "🎖️",
		Styled:      "🎖️ *𝕲𝖊𝖓𝖊𝖗𝖆𝖑*",
		Description: "Komandan garis depan yang mengarahkan pertempuran dengan keberanian.",
	},
	{
		Name:        "Diplomat",
		MinChips:    100000,
		Emoji:       "⚖️",
		Styled:      "⚖️ *𝓓𝓲𝓹𝓵𝓸𝓶𝓪𝓽*",
		Description: "Negosiator ulung yang membangun jembatan melalui ketajaman logika dan tutur kata.",
	},
	{
		Name:        "Governor",
		MinChips:    50000,
		Emoji:       "📜",
		Styled:      "📜 *𝘎𝘰𝘷𝘦𝘳𝘯𝘰𝘳*",
		Description: "Pemimpin yang menata kesejahteraan di wilayahnya.",
	},
	{
		Name:        "Mercenary",
		MinChips:    20000,
		Emoji:       "⚔️",
		Styled:      "⚔️ *𝗠𝗲𝗿𝗰𝗲𝗻𝗮𝗿𝘆*",
		Description: "Sosok pengembara yang menukar kemampuannya demi tujuan dan kehormatan.",
	},
	{
		Name:        "Levy",
		MinChips:    5000,
		Emoji:       "🛡️",
		Styled:      "🛡️ *𝖫𝖾𝗏𝗒*",
		Description: "Individu tangguh yang berdiri tegak membela kedaulatan tanah tempat ia berpijak.",
	},
	{
		Name:        "Peasant",
		MinChips:    0,
		Emoji:       "🪓",
		Styled:      "🪓 _Peasant_",
		Description: "Jiwa yang memulai perjalanan hidupnya dari titik nol.",
	},
}

// GetRankByBalance mencari pangkat pemain berdasarkan jumlah saldonya saat ini.
func GetRankByBalance(balance int) RankInfo {
	for _, r := range Ranks {
		if balance >= r.MinChips {
			return r
		}
	}
	return Ranks[len(Ranks)-1] // Fallback ke pangkat terendah (Peasant)
}
