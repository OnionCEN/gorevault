# 🦊 GoRevault

<p align="center">
  <a href="https://github.com/OnionCEN/gorevault">
  <img src="https://img.shields.io/github/stars/OnionCEN/gorevault?style=for-the-badge&logo=github&color=ff69b4&label=Stars" />
</a>
  <a href="https://discord.gg/fdThDK2Xgc">
    <img src="https://img.shields.io/badge/Discord-加入群聊-5865F2?style=for-the-badge&logo=discord&logoColor=white" />
  </a>
  <img src="https://img.shields.io/badge/QQ-158446355-12B7F5?style=for-the-badge&logo=tencentqq&logoColor=white" />
</p>

<p align="center">
  <i>os: 写代码好累，歇会儿 ☕</i>
</p>

---

<div align="center">

## 🧩 這是什麼？

就是把你电脑里的文件：

📄 切成一小块一小块
🔐 用密码锁起来
🧩 分散存到不同地方
🔄 想用的时候再拼回来

（*聽起來有點抽象對吧？其實就是防止你後悔。*）

</div>

---

<div align="center">

## 🤔 为啥要这么麻烦？

Because…

💾 硬盘会坏
💻 电脑会丢
🫠 手会抖（删错文件）
💔 论文会消失

</div>

---

# 🚀 下载 (Download)

```bash
git clone https://github.com/OnionCEN/gorevault.git
cd gorevault
go build -o gorevault cmd/gorevault/main.go
```

---

# 📦 怎么用

## 1️⃣ 第一次用

```bash
./gorevault init
```

执行完会多一个 `.gorevault` 文件夹。
👉 别删它，真的。

---

## 2️⃣ 备份文件

### 不加密

```bash
./gorevault backup -file 日记.txt
```

### 加密（怕被人偷看）

```bash
./gorevault backup -file 日记.txt -password "123456"
```

然后它会吐出来一串字符，那是**版本号**。
记下来，别随手关终端。

---

## 3️⃣ 查看历史

```bash
./gorevault version
```

会显示类似这样：

```
[1] a1b2c3d4
    时间: 2024-01-20 15:30
    文件: 日记.txt

[2] e5f6g7h8
    时间: 2024-01-21 10:20
    文件: 日记.txt
```

---

## 4️⃣ 恢复文件

### 恢复最新的

```bash
./gorevault restore -file 日记.txt
```

### 恢复某个版本

```bash
./gorevault restore -file 日记.txt -version a1b2c3d4
```

### 有密码的话

```bash
./gorevault restore -file 日记.txt -password "123456"
```

---

## 5️⃣ 和小伙伴互相备份（P2P模式）

电脑 A：

```bash
./gorevault p2p -port 4001
```

电脑 B：

```bash
./gorevault p2p -port 4002 -connect 电脑A的地址
```

然后你们就能互相看见对方的备份了。

（前提是网络别太离谱。）

---

# 📂 文件都放哪了？

```
.gorevault/
├── objects/    # 真正的数据块
├── versions/   # 版本记录
└── refs/       # 当前版本指针
```

没事别乱翻。
真翻坏了我也救不回来 🥲

---

# ⚠️ 已知问题

* 🔑 加密密码忘了就真的找不回来了（我也帮不了你）
* 🌐 P2P 需要公网IP或者内网穿透（学校宿舍可能不行）
* 🐢 第一次备份大文件会有点慢（要切块）

---

# 🛠 想改代码？

Fork 一份
改完发 Pull Request
我看心情合并 😼

---

# 💬 最后

写这个是因为我自己丢过论文。
真的很痛。

希望你别像我一样。

如果好用，给个 ⭐
不好用，提个 issue。
别骂我太狠，要不然我哭給你看qwq。
