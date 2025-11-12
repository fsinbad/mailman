#!/bin/bash

# Mailman K3s 一键部署脚本
# 用途: 在 K3s 集群中快速部署 Mailman 邮箱管理系统

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装，请先安装"
        exit 1
    fi
}

# 检查 K3s 是否运行
check_k3s() {
    if ! systemctl is-active --quiet k3s 2>/dev/null; then
        print_error "K3s 未运行，请先启动 K3s"
        print_info "安装 K3s: curl -sfL https://get.k3s.io | sh -"
        exit 1
    fi
}

# 等待 Pod 就绪
wait_for_pods() {
    local selector=$1
    local timeout=${2:-300}
    print_info "等待 Pod 就绪 (选择器: $selector, 超时: ${timeout}s)..."
    
    if kubectl wait --for=condition=ready pod -l "$selector" --timeout="${timeout}s" 2>/dev/null; then
        print_info "Pod 已就绪"
        return 0
    else
        print_warn "等待超时，但继续部署流程"
        return 1
    fi
}

# 显示欢迎信息
print_banner() {
    echo ""
    echo "╔═══════════════════════════════════════════╗"
    echo "║   Mailman K3s 部署脚本                    ║"
    echo "║   邮箱管理系统 - 轻量级 Kubernetes        ║"
    echo "╚═══════════════════════════════════════════╝"
    echo ""
}

# 检查前置条件
check_prerequisites() {
    print_info "检查前置条件..."
    
    check_command kubectl
    check_command helm
    check_k3s
    
    # 配置 kubectl
    if [ ! -f "$HOME/.kube/config" ]; then
        print_info "配置 kubectl..."
        mkdir -p ~/.kube
        sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
        sudo chown $(id -u):$(id -g) ~/.kube/config
    fi
    
    # 测试 kubectl 连接
    if ! kubectl get nodes &>/dev/null; then
        print_error "无法连接到 K3s 集群"
        exit 1
    fi
    
    print_info "前置条件检查完成 ✓"
}

# 部署 MariaDB
deploy_mariadb() {
    print_info "部署 MariaDB 数据库..."
    
    if kubectl get pod -l app=mariadb 2>/dev/null | grep -q Running; then
        print_warn "MariaDB 已存在，跳过部署"
        return
    fi
    
    kubectl apply -f ./helm/mailman/matrixdb-deployment.yaml
    
    # 等待 MariaDB 就绪
    wait_for_pods "app=mariadb" 300
    
    print_info "MariaDB 部署完成 ✓"
}

# 部署 Mailman 应用
deploy_mailman() {
    print_info "部署 Mailman 应用..."
    
    # 检查是否已安装
    if helm list | grep -q mailman; then
        print_warn "Mailman 已安装，执行升级..."
        helm upgrade mailman ./helm/mailman \
            --values ./helm/mailman/values-matrixdb-production.yaml
    else
        helm install mailman ./helm/mailman \
            --namespace default \
            --values ./helm/mailman/values-matrixdb-production.yaml
    fi
    
    # 等待 Pod 就绪
    wait_for_pods "app.kubernetes.io/name=mailman" 300
    
    print_info "Mailman 应用部署完成 ✓"
}

# 配置访问方式
configure_access() {
    print_info "配置访问方式..."
    
    # 方式1: NodePort (本地开发推荐)
    print_info "配置 NodePort 访问..."
    kubectl patch svc mailman-frontend -p '{"spec":{"type":"NodePort"}}' 2>/dev/null || true
    
    # 获取 NodePort
    sleep 3
    PORT=$(kubectl get svc mailman-frontend -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null)
    
    if [ -n "$PORT" ]; then
        print_info "NodePort 访问方式配置完成 ✓"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  🎉 部署成功！"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo "访问方式："
        echo "  📍 本地访问: http://localhost:$PORT"
        echo "  📍 内网访问: http://$(hostname -I | awk '{print $1}'):$PORT"
        echo ""
    fi
    
    # 方式2: Ingress (可选)
    print_info "提示: 如需配置域名访问，请参考 K3S_DEPLOYMENT.md 文档"
}

# 显示部署状态
show_status() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  📊 部署状态"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    print_info "Pods 状态:"
    kubectl get pods -l app.kubernetes.io/name=mailman -o wide
    kubectl get pods -l app=mariadb -o wide
    
    echo ""
    print_info "Services 状态:"
    kubectl get svc -l app.kubernetes.io/name=mailman
    kubectl get svc -l app=mariadb
    
    echo ""
    print_info "PVC 状态:"
    kubectl get pvc
    
    echo ""
}

# 显示有用的命令
show_useful_commands() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  🛠️  常用命令"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "查看日志:"
    echo "  kubectl logs -f deployment/mailman-backend"
    echo "  kubectl logs -f deployment/mailman-frontend"
    echo ""
    echo "重启服务:"
    echo "  kubectl rollout restart deployment/mailman-backend"
    echo "  kubectl rollout restart deployment/mailman-frontend"
    echo ""
    echo "查看详细状态:"
    echo "  kubectl get all -l app.kubernetes.io/name=mailman"
    echo ""
    echo "卸载应用:"
    echo "  helm uninstall mailman"
    echo "  kubectl delete -f ./helm/mailman/matrixdb-deployment.yaml"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
}

# 清理函数
cleanup() {
    print_warn "正在清理部署..."
    
    read -p "确定要卸载 Mailman 吗？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        helm uninstall mailman 2>/dev/null || true
        kubectl delete -f ./helm/mailman/matrixdb-deployment.yaml 2>/dev/null || true
        print_info "清理完成"
    else
        print_info "取消清理"
    fi
}

# 主函数
main() {
    print_banner
    
    # 解析参数
    case "${1:-}" in
        "cleanup"|"uninstall")
            cleanup
            exit 0
            ;;
        "status")
            show_status
            exit 0
            ;;
        "help"|"-h"|"--help")
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  (无参数)     - 执行完整部署"
            echo "  status       - 显示部署状态"
            echo "  cleanup      - 卸载部署"
            echo "  help         - 显示此帮助信息"
            echo ""
            exit 0
            ;;
    esac
    
    # 执行部署流程
    check_prerequisites
    deploy_mariadb
    deploy_mailman
    configure_access
    show_status
    show_useful_commands
    
    print_info "部署流程完成！"
}

# 执行主函数
main "$@"