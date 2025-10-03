# Release 配置说明

## GitHub Secrets 配置

在使用 GitHub Actions 自动发布前，需要在仓库设置中添加以下 Secrets：

1. 进入仓库的 `Settings` → `Secrets and variables` → `Actions`
2. 点击 `New repository secret` 添加以下密钥：

### 必需的 Secrets

| Secret 名称 | 说明 | 获取方式 |
|------------|------|---------|
| `DOCKERHUB_USERNAME` | Docker Hub 用户名 | 你的 Docker Hub 用户名 |
| `DOCKERHUB_TOKEN` | Docker Hub 访问令牌 | 在 Docker Hub → Account Settings → Security → New Access Token |

### GITHUB_TOKEN

`GITHUB_TOKEN` 是 GitHub 自动提供的，无需手动配置。

## 发布流程

### 1. 创建 Release Tag

```bash
# 打标签
git tag -a v1.0.0 -m "Release v1.0.0"

# 推送标签到 GitHub
git push origin v1.0.0
```

### 2. 自动构建

推送标签后，GitHub Actions 会自动：
- ✅ 编译 Linux (amd64/arm64)
- ✅ 编译 macOS (amd64/arm64)
- ✅ 编译 Windows (amd64)
- ✅ 构建 Docker 镜像（多架构：amd64/arm64）
- ✅ 推送到 Docker Hub
- ✅ 创建 GitHub Release 并上传所有文件

### 3. 查看结果

- **GitHub Release**: `https://github.com/你的用户名/UniBarrage/releases`
- **Docker Hub**: `https://hub.docker.com/r/你的用户名/unibarrage`

## Docker Hub Token 获取步骤

1. 登录 [Docker Hub](https://hub.docker.com/)
2. 点击右上角头像 → `Account Settings`
3. 左侧菜单选择 `Security`
4. 点击 `New Access Token`
5. 输入描述（如 `github-actions`）
6. 权限选择 `Read, Write, Delete`
7. 点击 `Generate`
8. **复制生成的 Token（只会显示一次）**
9. 将 Token 添加到 GitHub Secrets 中

## 版本号规范

推荐使用语义化版本号：
- `v1.0.0` - 主版本
- `v1.1.0` - 次版本（新功能）
- `v1.1.1` - 修订版（Bug 修复）

## 示例

```bash
# 开发完成后
git add .
git commit -m "feat: add new feature"
git push

# 准备发布
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 等待 GitHub Actions 完成（约 5-10 分钟）
# 完成后可在 GitHub Releases 页面查看
```
