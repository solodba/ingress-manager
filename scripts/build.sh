# 构建linux系统运行的golang程序
export GOOS=linux
export GOARCH=amd64
go build -a -o dist/ingress-manager

# 构建镜像
docker build -t huxiaodan/ingress-manager:v1.0.0 .
docker push huxiaodan/ingress-manager:v1.0.0