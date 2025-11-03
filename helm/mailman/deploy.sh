#!/bin/bash

# Mailman Helm Deployment Script
# This script helps deploy the Mailman application using Helm

set -e

# Default values
NAMESPACE="default"
RELEASE_NAME="mailman"
CHART_PATH="."
VALUES_FILE=""
ENVIRONMENT="development"
DEPLOYMENT_TYPE="standard"
CREATE_NAMESPACE=false
DRY_RUN=false
TIMEOUT=10m

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
usage() {
    cat << EOF
Mailman Helm Deployment Script

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --namespace NAMESPACE       Kubernetes namespace (default: default)
    -r, --release RELEASE_NAME      Helm release name (default: mailman)
    -e, --environment ENV           Environment (development|production) (default: development)
    -t, --type TYPE                 Deployment type (standard|allinone) (default: standard)
    -f, --values VALUES_FILE        Custom values file
    -c, --create-namespace          Create namespace if it doesn't exist
    -d, --dry-run                   Perform a dry run installation
    -T, --timeout TIMEOUT           Installation timeout (default: 10m)
    -h, --help                      Show this help message

EXAMPLES:
    # Standard development deployment
    $0 -n mailman-dev -e development -t standard

    # All-in-One development deployment (simplified)
    $0 -n mailman-allinone -e development -t allinone

    # Production deployment with custom values
    $0 -n mailman-prod -e production -t standard -f custom-values.yaml

    # Production all-in-one deployment
    $0 -n mailman-prod -e production -t allinone

    # Dry run to test installation
    $0 -n mailman-test -d

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -r|--release)
            RELEASE_NAME="$2"
            shift 2
            ;;
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -t|--type)
            DEPLOYMENT_TYPE="$2"
            shift 2
            ;;
        -f|--values)
            VALUES_FILE="$2"
            shift 2
            ;;
        -c|--create-namespace)
            CREATE_NAMESPACE=true
            shift
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -T|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(development|production)$ ]]; then
    print_error "Environment must be either 'development' or 'production'"
    exit 1
fi

# Validate deployment type
if [[ ! "$DEPLOYMENT_TYPE" =~ ^(standard|allinone)$ ]]; then
    print_error "Deployment type must be either 'standard' or 'allinone'"
    exit 1
fi

# Set values file based on environment and deployment type if not provided
if [[ -z "$VALUES_FILE" ]]; then
    if [[ "$DEPLOYMENT_TYPE" == "allinone" ]]; then
        VALUES_FILE="values-all-in-one.yaml"
    else
        VALUES_FILE="values-${ENVIRONMENT}.yaml"
    fi
fi

# Check if values file exists
if [[ ! -f "$VALUES_FILE" ]]; then
    print_error "Values file not found: $VALUES_FILE"
    exit 1
fi

print_status "Deploying Mailman with the following configuration:"
echo "  Namespace: $NAMESPACE"
echo "  Release Name: $RELEASE_NAME"
echo "  Environment: $ENVIRONMENT"
echo "  Deployment Type: $DEPLOYMENT_TYPE"
echo "  Values File: $VALUES_FILE"
echo "  Create Namespace: $CREATE_NAMESPACE"
echo "  Dry Run: $DRY_RUN"
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    print_error "helm is not installed or not in PATH"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster"
    exit 1
fi

# Create namespace if requested
if [[ "$CREATE_NAMESPACE" == true ]]; then
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        print_status "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
        print_success "Namespace created: $NAMESPACE"
    else
        print_warning "Namespace already exists: $NAMESPACE"
    fi
fi

# Build Helm command
HELM_CMD="helm upgrade --install $RELEASE_NAME $CHART_PATH"
HELM_CMD="$HELM_CMD --namespace $NAMESPACE"
HELM_CMD="$HELM_CMD --values $VALUES_FILE"
HELM_CMD="$HELM_CMD --timeout $TIMEOUT"

# Add dry run flag if requested
if [[ "$DRY_RUN" == true ]]; then
    HELM_CMD="$HELM_CMD --dry-run --debug"
    print_status "Performing dry run installation..."
fi

# Add namespace flag if creating namespace
if [[ "$CREATE_NAMESPACE" == true ]]; then
    HELM_CMD="$HELM_CMD --create-namespace"
fi

# For production environment, add additional flags
if [[ "$ENVIRONMENT" == "production" ]]; then
    HELM_CMD="$HELM_CMD --wait"
    print_status "Production deployment: waiting for all resources to be ready..."
fi

print_status "Executing Helm command:"
echo "$HELM_CMD"
echo ""

# Execute Helm command
if eval "$HELM_CMD"; then
    if [[ "$DRY_RUN" == true ]]; then
        print_success "Dry run completed successfully"
    else
        print_success "Mailman deployed successfully!"

        # Show deployment status
        print_status "Checking deployment status..."
        kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=mailman"

        # Show services
        print_status "Services:"
        kubectl get services -n "$NAMESPACE" -l "app.kubernetes.io/name=mailman"

        # Show ingress if enabled
        if kubectl get ingress -n "$NAMESPACE" -l "app.kubernetes.io/name=mailman" &> /dev/null; then
            print_status "Ingress:"
            kubectl get ingress -n "$NAMESPACE" -l "app.kubernetes.io/name=mailman"
        fi

        print_status "Deployment information:"
        echo "  Release: $RELEASE_NAME"
        echo "  Namespace: $NAMESPACE"
        echo "  Environment: $ENVIRONMENT"
        echo ""
        echo "To check the status:"
        echo "  helm status $RELEASE_NAME -n $NAMESPACE"
        echo ""
        echo "To uninstall:"
        echo "  helm uninstall $RELEASE_NAME -n $NAMESPACE"
    fi
else
    print_error "Deployment failed!"
    exit 1
fi