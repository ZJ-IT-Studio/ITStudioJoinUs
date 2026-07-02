package server

type SiteContent struct {
	HeroEyebrow     string        `json:"heroEyebrow"`
	HeroTitle       string        `json:"heroTitle"`
	HeroSubtitle    string        `json:"heroSubtitle"`
	ManifestoTitle  string        `json:"manifestoTitle"`
	ManifestoBody   string        `json:"manifestoBody"`
	DirectionsTitle string        `json:"directionsTitle"`
	Directions      []ContentCard `json:"directions"`
	Values          []ContentCard `json:"values"`
	Process         []ContentCard `json:"process"`
	FAQs            []ContentCard `json:"faqs"`
	ContactTitle    string        `json:"contactTitle"`
	ContactBody     string        `json:"contactBody"`
	ContactLink     string        `json:"contactLink"`
}
type ContentCard struct {
	Label string `json:"label"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func DefaultContent() SiteContent {
	return SiteContent{
		HeroEyebrow: "IT STUDIO / CAMPUS RECRUITMENT 2026", HeroTitle: "BUILD WHAT'S NEXT", HeroSubtitle: "把想法编译成现实。这里不缺观众，缺的是下一位共创者。",
		ManifestoTitle: "不是等待未来，\n是亲手构建它。", ManifestoBody: "IT Studio 是一群对技术、设计和创造保持好奇的人。我们在真实项目中学习，也让每一次试错都留下可以运行的答案。",
		DirectionsTitle: "CHOOSE YOUR VECTOR / 选择你的坐标",
		Directions:      []ContentCard{{"01 / DEV", "开发", "从第一行代码到稳定上线。"}, {"02 / DESIGN", "设计", "让复杂的事情变得清楚、好用、有记忆点。"}, {"03 / PRODUCT", "产品", "发现真实问题，推动想法成为作品。"}, {"04 / OPS", "运营", "连接人、内容与现场，让好项目被看见。"}},
		Values:          []ContentCard{{"MAKE", "在做中学", "真实项目、真实协作、真实反馈。"}, {"SHARE", "开放共享", "把踩过的坑变成下一位同伴的路标。"}, {"SHIP", "交付作品", "灵感很珍贵，能运行的灵感更加珍贵。"}},
		Process:         []ContentCard{{"01", "在线报名", "选择方向，告诉我们你想创造什么。"}, {"02", "作品阅读", "我们会认真看完每一份表达。"}, {"03", "轻松面谈", "聊聊兴趣、经历和彼此的期待。"}, {"04", "共同出发", "加入小组，从第一个项目开始。"}},
		FAQs:            []ContentCard{{"Q1", "没有基础可以报名吗？", "可以。我们更关心你的好奇心、投入和学习能力。"}, {"Q2", "可以跨专业报名吗？", "当然。方向不是边界，真实问题也从不按专业出现。"}, {"Q3", "报名后如何查看进度？", "使用学号和查询密码进入报名中心即可查看。"}},
		ContactTitle:    "READY TO COMPILE?", ContactBody: "下一次刷新页面时，也许你已经在这里留下了自己的作品。", ContactLink: "mailto:itstudio@example.com",
	}
}
