package endpoints

// 各平台上游 API 与 CDN 基址常量，便于统一维护与扩展。

const (
	DouyinUserPageBase   = "https://www.douyin.com/user/self"              // 抖音用户页（modal_id 方案）
	DouyinIesShareBase   = "https://www.iesdouyin.com/share/video/"        // iesdouyin 分享页（No Cookie 方案）
	DouyinAwemeDetailAPI = "https://www.douyin.com/aweme/v1/web/aweme/detail/"
	DouyinPlayBase       = "https://aweme.snssdk.com/aweme/v1/play/" // 抖音播放基址

	KuaishouCDNVideo = "http://tx2.a.yximgs.com/"   // 快手CDN视频基址
	KuaishouCDNMusic = "http://txmov2.a.kwimgs.com" // 快手CDN音乐基址

	BilibiliViewAPI    = "https://api.bilibili.com/x/web-interface/view" // 哔哩哔哩视图API
	BilibiliPlayURLAPI = "https://api.bilibili.com/x/player/playurl"     // 哔哩哔哩播放URLAPI
	BilibiliMirrorCDN  = "https://upos-sz-mirrorhw.bilivideo.com/"       // 哔哩哔哩镜像CDN基址

	XHSItemAPI  = "https://www.xiaohongshu.com/discovery/item/" // 小红书itemAPI
	XHSImageCDN = "https://sns-img-hw.xhscdn.com/"              // 小红书图片CDN基址
	XHSVideoCDN = "http://sns-video-bd.xhscdn.com/"             // 小红书视频CDN基址
	XHSCoverCDN = "https://ci.xiaohongshu.com/"                 // 小红书封面CDN基址

	ToutiaoVideoPage = "https://www.toutiao.com/video/" // 头条视频页基址

	WeiboTVAPI   = "https://weibo.com/tv/api/component" // 微博TVAPI
	WeiboReferer = "https://weibo.com/"                 // 微博Referer
	WeiboOrigin  = "https://weibo.com"                  // 微博Origin

	DoubaoShareAPI = "https://www.doubao.com/creativity/share/get_video_share_info?version_code=20800&language=zh-CN&device_platform=web&aid=497858&real_aid=497858&pkg_type=release_version&device_id=&pc_version=3.8.3&region=&sys_region=&samantha_web=1&use-olympus-account=1&web_tab_id=4d5d17a6-6729-4c1e-9f55-09f093100f0a" // 豆包分享API
	DoubaoOrigin   = "https://www.doubao.com"                                                                                                                                                                                                                                                                                      // 豆包Origin
)
