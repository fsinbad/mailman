// API Response Types
export interface ApiResponse<T> {
    data: T;
    error?: string;
    message?: string;
}

// Email Account Types - 匹配后端API响应
export interface EmailAccount {
    id: number;
    emailAddress: string;
    authType: 'password' | 'oauth2' | 'app_password';
    password?: string;
    token?: string;
    mailProviderId: number;
    mailProvider?: {
        id: number;
        name: string;
        type: string;
        imapServer: string;
        imapPort: number;
        smtpServer: string;
        smtpPort: number;
        createdAt: string;
        updatedAt: string;
        deletedAt?: {
            time: string;
            valid: boolean;
        };
    };
    proxy?: string;
    isDomainMail: boolean;
    domain?: string;
    customSettings?: Record<string, any>;
    isVerified?: boolean;
    verifiedAt?: string;
    createdAt: string;
    updatedAt: string;
    deletedAt?: {
        time: string;
        valid: boolean;
    };
    // 前端添加的字段
    status?: 'active' | 'inactive' | 'error';
    lastSync?: string;
}

// 前端显示用的简化类型
export interface EmailAccountDisplay {
    id: number;
    email: string;
    provider: string;
    auth_type: 'password' | 'oauth2' | 'app_password';
    username?: string;
    status: 'active' | 'inactive' | 'error';
    last_sync?: string;
    use_proxy?: boolean;
    proxy_url?: string;
    proxy_username?: string;
    created_at: string;
    updated_at: string;
}

export interface CreateEmailAccountRequest {
    email_address: string;
    password?: string;
    auth_type?: 'password' | 'oauth2' | 'app_password';
    app_password?: string;
    token?: string;
    mail_provider_id?: number;
    proxy?: string;
    is_domain_mail?: boolean;
    domain?: string;
    custom_settings?: Record<string, any>;
}

export interface UpdateEmailAccountRequest extends Partial<CreateEmailAccountRequest> {
    id: number;
}

// Email Types - 匹配后端API响应
export interface Email {
    ID: number;
    MessageID: string;
    AccountID: number;
    Subject: string;
    From: string[];
    To: string[];
    Cc?: string[] | null;
    Bcc?: string[] | null;
    Date: string;
    Body: string;
    HTMLBody?: string;
    RawMessage?: string; // 原始邮件报文
    Attachments?: EmailAttachment[] | null;
    MailboxName: string;
    Flags?: string[];
    Size: number;
    CreatedAt: string;
    UpdatedAt: string;
    DeletedAt?: {
        time: string;
        valid: boolean;
    } | null;
    Account?: any;
}

export interface EmailAttachment {
    id: number;
    email_id: number;
    filename: string;
    content_type: string;
    size: number;
    content?: string;
}

export interface Attachment {
    id: number;
    email_id: number;
    filename: string;
    content_type: string;
    size: number;
    content?: string;
}

// Fetch Emails Types
export interface FetchEmailsRequest {
    folder?: string;
    limit?: number;
    since_date?: string;
    before_date?: string;
    subject_filter?: string;
    sender_filter?: string;
    unread_only?: boolean;
    with_attachments_only?: boolean;
    mark_as_read?: boolean;
    delete_after_fetch?: boolean;
    use_incremental_sync?: boolean;
    extract_content?: ExtractContentConfig;
}

export interface ExtractContentConfig {
    patterns?: ExtractPattern[];
}

export interface ExtractPattern {
    name: string;
    type: 'regex' | 'javascript' | 'go_template';
    pattern: string;
    flags?: string;
}

// Mail Provider Types
export interface MailProvider {
    id: number;
    name: string;
    imap_host: string;
    imap_port: number;
    imap_use_ssl: boolean;
    smtp_host?: string;
    smtp_port?: number;
    smtp_use_ssl?: boolean;
    oauth2_enabled: boolean;
    oauth2_auth_url?: string;
    oauth2_token_url?: string;
    oauth2_client_id?: string;
    oauth2_scopes?: string[];
}

// Wait Email Types
export interface WaitEmailRequest {
    email?: string;
    provider?: string;
    password?: string;
    app_password?: string;
    timeout?: number;
    folder?: string;
    subject_contains?: string;
    from_contains?: string;
    to_contains?: string;
    extract_content?: ExtractContentConfig;
}

// Statistics Types
export interface EmailStatistics {
    total_emails: number;
    unread_emails: number;
    today_emails: number;
    accounts_count: number;
    active_accounts: number;
    last_sync_time?: string;
}

// Sync Types
export interface SyncStatus {
    account_id: number;
    status: 'idle' | 'syncing' | 'error';
    progress?: number;
    message?: string;
    last_sync?: string;
    emails_fetched?: number;
}

// Filter Types
export interface EmailFilter {
    account_id?: number;
    folder?: string;
    search?: string;
    unread_only?: boolean;
    has_attachments?: boolean;
    date_from?: string;
    date_to?: string;
    sender?: string;
    subject?: string;
}

// Pagination Types
export interface PaginationParams {
    page?: number;
    limit?: number;
    sort_by?: string;
    sort_order?: 'asc' | 'desc';
    search?: string;  // 添加搜索字段，用于邮箱地址模糊查询
}

export interface PaginatedResponse<T> {
    data: T[];
    total: number;
    page: number;
    limit: number;
    total_pages: number;
}

// 转换函数：将API响应转换为前端显示格式
export function convertToDisplayAccount(account: EmailAccount): EmailAccountDisplay {
    return {
        id: account.id,
        email: account.emailAddress,
        provider: account.mailProvider?.name || account.mailProvider?.type || 'Unknown',
        auth_type: account.authType,
        status: 'active', // 默认状态，可以根据其他字段判断
        last_sync: account.lastSync,
        use_proxy: !!account.proxy,
        proxy_url: account.proxy,
        created_at: account.createdAt,
        updated_at: account.updatedAt,
    };
}

// 取件模板相关类型
export interface ExtractorConfig {
    field: 'ALL' | 'from' | 'to' | 'cc' | 'subject' | 'body' | 'html_body' | 'headers'
    type: 'regex' | 'js' | 'gotemplate'
    match?: string  // 可选的匹配条件
    extract: string // 提取规则（替换原来的config字段）
    config?: string // 保留用于向后兼容
    replacement?: string // 正则表达式的替换模板（如 $0, $1 等）
}

export interface ExtractResult {
    field: string
    value: string
    confidence?: number
}

export interface ExtractorTemplate {
    id: number
    name: string
    description?: string
    extractors: ExtractorConfig[]
    created_at: string
    updated_at: string
}

export interface ExtractorTemplateRequest {
    name: string
    description?: string
    extractors: ExtractorConfig[]
}

export interface PaginatedExtractorTemplatesResponse {
    data: ExtractorTemplate[]
    total: number
    page: number
    limit: number
    total_pages: number
}

// 触发器相关类型
export type TriggerStatus = 'enabled' | 'disabled'
export type TriggerActionType = 'modify_content' | 'smtp'
export type TriggerExecutionStatus = 'success' | 'failed' | 'partial'

export interface TriggerConditionConfig {
    type: string // js, gotemplate
    script: string // 脚本内容
    timeout?: number // 超时时间（秒）
}

export interface TriggerActionConfig {
    type: TriggerActionType // 动作类型
    name: string // 动作名称
    description?: string // 动作描述
    config: string // 动作配置（JSON字符串或模板）
    enabled: boolean // 是否启用此动作
    order: number // 执行顺序
}

export interface EmailTrigger {
    id: number
    name: string // 触发器名称
    description?: string // 触发器描述
    status: TriggerStatus // 触发器状态

    // 检查配置
    check_interval: number // 检查间隔（秒）

    // 过滤参数（复用EmailFilter结构）
    email_address?: string // 邮箱地址过滤
    start_date?: string // 开始日期
    end_date?: string // 结束日期
    subject?: string // 主题过滤
    from?: string // 发件人过滤
    to?: string // 收件人过滤
    has_attachment?: boolean // 是否有附件
    unread?: boolean // 是否未读
    labels?: string[] // 标签过滤
    folders?: string[] // 文件夹列表
    custom_filters?: Record<string, string> // 自定义过滤器

    // 触发条件和动作
    condition: TriggerConditionConfig // 触发条件
    actions: TriggerActionConfig[] // 触发动作

    // 日志配置
    enable_logging: boolean // 是否启用日志

    // 统计信息
    total_executions: number // 总执行次数
    success_executions: number // 成功执行次数
    last_executed_at?: string // 最后执行时间
    last_error?: string // 最后错误信息

    // 时间戳
    created_at: string
    updated_at: string
    deleted_at?: string
}

export interface CreateTriggerRequest {
    name: string
    description?: string
    check_interval: number
    email_address?: string
    subject?: string
    from?: string
    to?: string
    has_attachment?: boolean
    unread?: boolean
    labels?: string[]
    folders?: string[]
    custom_filters?: Record<string, string>
    condition: TriggerConditionConfig
    actions: TriggerActionConfig[]
    enable_logging: boolean
    status: TriggerStatus
}

export interface UpdateTriggerRequest extends Partial<CreateTriggerRequest> {
    id: number
}

export interface TriggerActionResult {
    action_name: string
    action_type: string
    success: boolean
    error?: string
    input_data?: any
    output_data?: any
    execution_ms: number
}

export interface TriggerExecutionLog {
    id: number
    trigger_id: number
    trigger?: EmailTrigger

    // 执行信息
    status: TriggerExecutionStatus
    start_time: string
    end_time: string
    execution_ms: number

    // 输入参数
    email_id: number
    email?: Email
    input_params?: Record<string, any> // 触发器入口参数

    // 条件校验结果
    condition_result: boolean
    condition_error?: string

    // 动作执行结果
    action_results: TriggerActionResult[]

    // 错误信息
    error_message?: string

    // 时间戳
    created_at: string
}

export interface PaginatedTriggersResponse {
    data: EmailTrigger[]
    total: number
    page: number
    limit: number
    total_pages: number
}

export interface PaginatedTriggerLogsResponse {
    data: TriggerExecutionLog[]
    total: number
    page: number
    limit: number
    total_pages: number
}

export interface TriggerStatistics {
    total_executions: number
    success_executions: number
    failed_executions: number
    partial_executions: number
    avg_execution_time: number
    success_rate: number
    max_execution_time: number
    min_execution_time: number
    avg_condition_time: number
    avg_action_time: number
    execution_time_percentiles: {
        p50: number
        p90: number
        p95: number
        p99: number
    }
    resource_usage?: {
        avg_memory_mb: number
        max_memory_mb: number
        avg_cpu_percent: number
        max_cpu_percent: number
    }
    time_distribution?: {
        labels: string[]
        values: number[]
    }
    executions_by_day?: {
        dates: string[]
        counts: number[]
        success_counts: number[]
        failed_counts: number[]
    }
}

// OAuth2 Types
export type OAuth2ProviderType = 'gmail' | 'outlook'

export interface OAuth2GlobalConfig {
    id: number
    name: string
    provider_type: OAuth2ProviderType
    client_id: string
    client_secret: string
    redirect_uri: string
    scopes: string[]
    is_enabled: boolean
    created_at: string
    updated_at: string
}

export interface CreateOAuth2ConfigRequest {
    name: string
    provider_type: OAuth2ProviderType
    client_id: string
    client_secret: string
    redirect_uri: string
    scopes: string[]
    is_enabled: boolean
}

export interface UpdateOAuth2ConfigRequest extends Partial<CreateOAuth2ConfigRequest> {
    id: number
}

export interface OAuth2AuthUrlRequest {
    provider: OAuth2ProviderType
    state?: string
}

export interface OAuth2AuthUrlResponse {
    auth_url: string
    state: string
}

export interface OAuth2TokenExchangeRequest {
    provider: OAuth2ProviderType
    code: string
    state: string
    config_id?: number  // Optional: specific OAuth2 config to use
}

export interface OAuth2TokenResponse {
    access_token: string
    refresh_token?: string
    token_type: string
    expires_in: number
    scope?: string
}

export interface OAuth2RefreshTokenRequest {
    provider: OAuth2ProviderType
    refresh_token: string
    config_id?: number  // Optional: specific OAuth2 config to use
}


export interface OAuth2AuthUrlRequest {
    provider: OAuth2ProviderType
    redirect_uri?: string
}

export interface OAuth2AuthUrlResponse {
    auth_url: string
    state: string
}


// OAuth2 Account Integration
export interface OAuth2AccountInfo {
    email: string
    name?: string
    provider: OAuth2ProviderType
    access_token: string
    refresh_token: string
    expires_at: number
}

export interface CreateOAuth2AccountRequest {
    email_address: string
    provider: OAuth2ProviderType
    access_token: string
    refresh_token: string
    expires_at: number
}