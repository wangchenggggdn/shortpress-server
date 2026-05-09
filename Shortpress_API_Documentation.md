# Shortpress API Documentation

## 概览

Shortpress 是一个为内容创作者设计的视频网站构建平台。该平台使创作者能够轻松构建自己的独立视频网站，具有直观的界面和简化的流程，让创作者能够上传、管理和展示他们的视频，无需复杂的技术知识。

## 基本信息

- **Base URL**: `http://192.168.6.160:8080/api`
- **认证方式**: 通过请求头 `X-Site-Id` 进行站点识别
- **数据格式**: JSON
- **字符编码**: UTF-8

## 通用响应格式

所有API响应都遵循以下格式：

```json
{
  "code": 0,        // 状态码，0表示成功，其他值表示错误
  "info": "ok",     // 状态信息
  "data": {}        // 具体数据内容
}
```

## 通用请求头

大部分请求都应包含以下基本头信息

```http
Content-Type: application/json
X-Site-Id: {站点ID}
X-App-Name: {应用名称}
X-App-Version: 1.0.0
User-Agent: {用户代理信息}
```

## API 调用流程
当前文档接口全部是向C端用户提供的API，
1. 当前这套后端 API 会支持多个站点，所有基本每一个接口都有一个 siteId 的概念。
2. 站点进来第一步是要先使用 /api/client/site/info 接口通过当前用户的 domain 或者 sitePath 查询站点信息，并给出siteID供其他接口使用
3. 用户注册登录返回的 token 前端页面需要存储，登录以后的接口请求要将其要带到 Authorization 中。

---

## 1. 站点管理

### 1.1 获取站点信息

获取指定站点的基本信息和配置。
一般来说 C 端用户请求站点信息有一下几种方式，假如B端用户配置的站点的path是 abc, 那么通过浏览器地址输入的
1. http://abc.myshortpress.com 其中 myshortpress.com是官方地址
2. http://xxx.your-custom-domain 这种为自定义
3. 目前path方式会被忽略，以上1和2都是可以通过传递domain实现

**接口地址**: `GET /api/client/site/info`

**请求参数**:
- `domain` (string, required): 域名 ，优先使用, sitePath 查询站点信息
- `sitePath` (string, required): 站点路径， 优先使用 sitePath 查询站点信息。

**示例请求**:
```http
GET /api/client/site/info?domain=myshortpress.com&sitePath=fruit0007
```

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "siteId": "29022543-b83e-42f4-8567-13b2f78f642c",
    "officialDomain": "fruit0007.myshortpress.com",
    "domain": "balabala.work",
    "redirect": false,
    "path": "fruit0007",
    "name": "MySite Updated",
    "logo": "http://192.168.6.160:8080/res/img/7b6108f0bcfd6aad4d234b994503b676/0518a758-d553-40c8-99fe-be369e0d7af8.jpg",
    "googleAnalyticsId": "aaaaaa-xxxx",
    "facebookPixelId": "jjjj",
    "thinkingdataAppId": null,
    "status": 0,
    "seo": {
      "title": "BBBB to MySite Updated",
      "description": "BBB  is my updated website",
      "keywords": "site11BBB,example,updated"
    }
  }
}
```

**字段说明**:
- `siteId`: 站点唯一标识符
- `officialDomain`: 官方域名
- `domain`: 自定义域名
- `redirect`: 是否重定向
- `path`: 站点路径
- `name`: 站点名称
- `logo`: 站点logo图片URL
- `googleAnalyticsId`: Google Analytics ID
- `facebookPixelId`: Facebook Pixel ID
- `thinkingdataAppId`: ThinkingData应用ID
- `status`: 站点状态 (0: 正常)
- `seo`: SEO配置信息

### 1.2 获取站点页面配置

获取站点的页面构建配置信息。

**接口地址**: `GET /api/client/site/pages`

**请求参数**:
- `sitePath` (string): 站点路径

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "pages": [],
    "config": {}
  }
}
```

---

## 2. 播放列表管理

### 2.1 获取播放列表信息

获取指定播放列表的详细信息。

**接口地址**: `GET /api/client/playlist/info`

**请求参数**:
- `playlistId` (string, required): 播放列表ID
- `needVid` (boolean): 是否需要包含视频列表 (true/false)

**示例请求**:
```http
GET /api/client/playlist/info?needVid=false&playlistId=47911be5-6558-4afb-8eaa-7c08b0509e47
```

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "playlistId": "47911be5-6558-4afb-8eaa-7c08b0509e47",
    "title": "004",
    "slug": "004",
    "description": "004",
    "tags": "",
    "cover": "http://192.168.6.160:8080/res/img/7b6108f0bcfd6aad4d234b994503b676/815d08a7-8eb3-4615-ae46-27466d224970.jpg",
    "status": 0,
    "videoCount": 9,
    "version": 0,
    "accessType": 0,
    "singleVideoPrice": 0,
    "freeVideos": 0,
    "seo": null,
    "videos": null,
    "createdAt": 0,
    "updatedAt": 0
  }
}
```

**字段说明**:
- `playlistId`: 播放列表唯一标识符
- `title`: 播放列表标题
- `slug`: URL友好的标识符
- `description`: 描述信息
- `tags`: 标签
- `cover`: 封面图片URL
- `status`: 状态 (0: 正常)
- `videoCount`: 视频数量
- `version`: 版本号
- `accessType`: 访问类型 (0: 免费)
- `singleVideoPrice`: 单个视频价格
- `freeVideos`: 免费视频数量
- `seo`: SEO配置
- `videos`: 视频列表 (当needVid=true时包含)

### 2.2 获取播放列表中的视频

获取指定播放列表中的视频列表。

**接口地址**: `GET /api/client/playlist/videos`

**请求参数**:
- `playlistId` (string, required): 播放列表ID

**示例请求**:
```http
GET /api/client/playlist/videos?playlistId=47911be5-6558-4afb-8eaa-7c08b0509e47
```

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "total": 9,
    "page": 0,
    "pageSize": 0,
    "hasMore": false,
    "items": [
      {
        "vid": "21b3d98a-4c4f-4b93-b272-207a503a11f8",
        "status": 2,
        "unlockStatus": 1
      },
      {
        "vid": "4a50ae2a-1eef-48e6-93e3-12d69001ec99",
        "status": 2,
        "unlockStatus": 1
      }
    ]
  }
}
```

**字段说明**:
- `total`: 总数量
- `page`: 当前页码
- `pageSize`: 每页大小
- `hasMore`: 是否有更多数据
- `items`: 视频项目列表
  - `vid`: 视频ID
  - `status`: 视频状态 (2: 正常)
  - `unlockStatus`: 解锁状态 (1: 已解锁)

### 2.3 批量获取播放列表

**接口地址**: `GET /api/playlist/batch-get`

**请求参数**:
- `pids` (string): 播放列表ID列表，用逗号分隔

---

## 3. 视频管理

### 3.1 批量获取视频信息

根据视频ID列表批量获取视频详细信息。

**接口地址**: `GET /api/video/batch-get`

**请求参数**:
- `vids` (string, required): 视频ID列表，URL编码后用逗号分隔

**示例请求**:
```http
GET /api/video/batch-get?vids=21b3d98a-4c4f-4b93-b272-207a503a11f8%2C4a50ae2a-1eef-48e6-93e3-12d69001ec99
```

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "total": 0,
    "page": 0,
    "pageSize": 0,
    "items": [
      {
        "vid": "21b3d98a-4c4f-4b93-b272-207a503a11f8",
        "title": "Download (48)",
        "description": "",
        "tags": "",
        "cover": "http://192.168.6.160:8080/videolib/7b6108f0bcfd6aad4d234b994503b676/21b3d98a-4c4f-4b93-b272-207a503a11f8.jpg",
        "duration": 73,
        "width": 576,
        "height": 1024,
        "status": 2,
        "uploadStatus": 5,
        "createdAt": 1746623563,
        "updatedAt": 1751283878,
        "fileSize": 0,
        "videoPath": "http://192.168.6.160:8080/videolib/7b6108f0bcfd6aad4d234b994503b676/21b3d98a-4c4f-4b93-b272-207a503a11f8.mp4",
        "videoSourceUrl": "http://192.168.6.160:8080/videolib/7b6108f0bcfd6aad4d234b994503b676/21b3d98a-4c4f-4b93-b272-207a503a11f8.mp4",
        "subtitles": null,
        "seo": null
      }
    ]
  }
}
```

**字段说明**:
- `vid`: 视频唯一标识符
- `title`: 视频标题
- `description`: 视频描述
- `tags`: 标签
- `cover`: 封面图片URL
- `duration`: 视频时长(秒)
- `width`: 视频宽度
- `height`: 视频高度
- `status`: 视频状态 (2: 正常)
- `uploadStatus`: 上传状态 (5: 上传完成)
- `createdAt`: 创建时间戳
- `updatedAt`: 更新时间戳
- `fileSize`: 文件大小
- `videoPath`: 视频播放路径
- `videoSourceUrl`: 视频源URL
- `subtitles`: 字幕信息
- `seo`: SEO配置

---

## 4. 内容推送与数据流

### 4.1 获取内容流

获取站点的内容推送流。

**接口地址**: `GET /api/client/feed`

**请求参数**:
- `page` (int): 页码
- `pageSize` (int): 每页大小
- `sitePath` (string): 站点路径

**示例请求**:
```http
GET /api/client/feed?page=2&pageSize=10&sitePath=
```

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "total": 0,
    "page": 0,
    "pageSize": 0,
    "hasMore": false,
    "items": []
  }
}
```

---

## 5. 用户管理

### 5.1 用户注册

注册新用户账户。

**接口地址**: `POST /api/user/register`

**请求体**:
```json
{
  "email": "user@example.com",
  "password": "password123",
  "confirmPassword": "password123"
}
```

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "userId": "c15066f8-d89b-4bae-8d5e-d6c559aa04be",
    "email": "user@example.com",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 5.2 获取用户资料

获取当前用户的资料信息。

**接口地址**: `GET /api/user/profile`

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "userId": "c15066f8-d89b-4bae-8d5e-d6c559aa04be",
    "email": "user@example.com",
    "profile": {
      "nickname": "用户昵称",
      "avatar": "头像URL",
      "bio": "个人简介"
    }
  }
}
```

---

## 6. 支付与代币系统

### 6.1 获取代币余额

获取用户的代币余额信息。

**接口地址**: `GET /api/client/payment/coins/balance`

**示例请求**:
```http
GET /api/client/payment/coins/balance
```

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {
    "balance": 0,
    "totalEarned": 0,
    "totalSpent": 0,
    "totalRealMoneySpent": 0.00
  }
}
```

**字段说明**:
- `balance`: 当前余额
- `totalEarned`: 总收入
- `totalSpent`: 总支出
- `totalRealMoneySpent`: 实际货币支出总额

---

## 7. 播放记录

### 7.1 记录视频播放

记录用户的视频播放进度和状态。

**接口地址**: `POST /api/client/video/playback/records`

**请求体**:
```json
{
  "vid": "c2a6c041-7513-4fa3-86df-4f498729da73",
  "playlistId": "0be2c008-9bed-4f3a-b75d-6dbdf5923c29",
  "playlistTitle": "002 xxx A B CC a ._--",
  "videoTitle": "Download (22)",
  "episodeNumber": 1,
  "duration": 11,
  "progress": 4,
  "cover": "http://192.168.6.160:8080/res/img/7b6108f0bcfd6aad4d234b994503b676/83647eec-3cee-41d7-a9ef-823b9e1f63bf.jpg",
  "playlistSlug": "002-xxx-a-b-cc-a-"
}
```

**字段说明**:
- `vid`: 视频ID
- `playlistId`: 播放列表ID
- `playlistTitle`: 播放列表标题
- `videoTitle`: 视频标题
- `episodeNumber`: 集数
- `duration`: 视频总时长
- `progress`: 播放进度
- `cover`: 封面图片URL
- `playlistSlug`: 播放列表URL标识

**响应示例**:
```json
{
  "code": 0,
  "info": "ok",
  "data": {}
}
```

---

## 8. 广告管理

### 8.1 获取广告单元配置

获取指定位置的广告单元配置。

**接口地址**: `GET /api/ads/unit/conf`

**请求参数**:
- `location` (string, required): 广告位置 (如: "Playlist")

**示例请求**:
```http
GET /api/ads/unit/conf?location=Playlist
```

**响应示例**:
```json
{
  "code": 404,
  "info": "Not Found",
  "data": {}
}
```

**注意**: 当广告配置不存在时，返回404状态。

---

## 状态码说明

### HTTP状态码
- `200 OK`: 请求成功
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误

### 业务状态码 (response.code)
- `0`: 成功
- `404`: 资源不存在
- 其他非零值: 各种业务错误

## 错误处理

当API返回错误时，响应格式如下：

```json
{
  "code": 404,
  "info": "Not Found",
  "data": {}
}
```

## 数据类型说明

- **UUID**: 统一资源标识符，格式如 `29022543-b83e-42f4-8567-13b2f78f642c`
- **Timestamp**: Unix时间戳，如 `1746623563`
- **URL**: 完整的HTTP(S) URL地址
- **Status Code**: 数字状态码

## 注意事项

1. **认证**: 所有客户端API请求都需要提供有效的 `X-Site-Id` 头信息
2. **编码**: 查询参数中的特殊字符需要进行URL编码
3. **大数据响应**: 当响应数据过大时，系统会自动截断并在日志中标记 "response too large, truncated"
4. **分页**: 支持分页的接口会返回 `page`, `pageSize`, `hasMore` 等分页信息
5. **版本控制**: API版本通过 `X-App-Version` 头信息指定

## 示例客户端代码

### JavaScript/Node.js

```javascript
const headers = {
  'Content-Type': 'application/json',
  'X-Site-Id': '29022543-b83e-42f4-8567-13b2f78f642c',
  'X-App-Name': 'Shortpress Client',
  'X-App-Version': '1.0.0'
};

// 获取站点信息
async function getSiteInfo(domain, sitePath) {
  const url = `http://192.168.6.160:8080/api/client/site/info?domain=${encodeURIComponent(domain)}&sitePath=${sitePath}`;
  const response = await fetch(url, { headers });
  return response.json();
}

// 获取播放列表信息
async function getPlaylistInfo(playlistId, needVid = false) {
  const url = `http://192.168.6.160:8080/api/client/playlist/info?playlistId=${playlistId}&needVid=${needVid}`;
  const response = await fetch(url, { headers });
  return response.json();
}

// 批量获取视频信息
async function getBatchVideos(videoIds) {
  const vids = videoIds.join(',');
  const url = `http://192.168.6.160:8080/api/video/batch-get?vids=${encodeURIComponent(vids)}`;
  const response = await fetch(url, { headers });
  return response.json();
}
```

### Python

```python
import requests
import urllib.parse

class ShortpressClient:
    def __init__(self, base_url, site_id):
        self.base_url = base_url
        self.headers = {
            'Content-Type': 'application/json',
            'X-Site-Id': site_id,
            'X-App-Name': 'Shortpress Python Client',
            'X-App-Version': '1.0.0'
        }
    
    def get_site_info(self, domain, site_path):
        url = f"{self.base_url}/client/site/info"
        params = {
            'domain': domain,
            'sitePath': site_path
        }
        response = requests.get(url, headers=self.headers, params=params)
        return response.json()
    
    def get_playlist_info(self, playlist_id, need_vid=False):
        url = f"{self.base_url}/client/playlist/info"
        params = {
            'playlistId': playlist_id,
            'needVid': str(need_vid).lower()
        }
        response = requests.get(url, headers=self.headers, params=params)
        return response.json()
    
    def get_batch_videos(self, video_ids):
        url = f"{self.base_url}/video/batch-get"
        vids = ','.join(video_ids)
        params = {'vids': vids}
        response = requests.get(url, headers=self.headers, params=params)
        return response.json()

# 使用示例
client = ShortpressClient(
    'http://192.168.6.160:8080/api',
    '29022543-b83e-42f4-8567-13b2f78f642c'
)

site_info = client.get_site_info('169.254.200.198:3001', 'fruit0007')
```

---

*最后更新时间: 2025-08-08*
*API版本: 1.0.0*
