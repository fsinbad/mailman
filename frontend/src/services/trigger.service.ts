import { apiClient } from '@/lib/api-client'
import {
    EmailTrigger,
    CreateTriggerRequest,
    UpdateTriggerRequest,
    PaginatedTriggersResponse,
    TriggerExecutionLog,
    PaginatedTriggerLogsResponse,
    TriggerStatistics,
    PaginationParams,
    ApiResponse
} from '@/types'

export class TriggerService {
    private static instance: TriggerService
    private baseUrl = '/triggers'

    static getInstance(): TriggerService {
        if (!TriggerService.instance) {
            TriggerService.instance = new TriggerService()
        }
        return TriggerService.instance
    }

    // 获取触发器列表（分页）
    async getTriggers(params?: PaginationParams): Promise<PaginatedTriggersResponse> {
        const queryParams = new URLSearchParams()

        if (params?.page) queryParams.append('page', params.page.toString())
        if (params?.limit) queryParams.append('limit', params.limit.toString())
        if (params?.sort_by) queryParams.append('sort_by', params.sort_by)
        if (params?.sort_order) queryParams.append('sort_order', params.sort_order)
        if (params?.search) queryParams.append('search', params.search)

        const url = queryParams.toString() ? `${this.baseUrl}?${queryParams}` : this.baseUrl
        const response = await apiClient.get<ApiResponse<PaginatedTriggersResponse>>(url)
        return response.data
    }

    // 获取单个触发器
    async getTrigger(id: number): Promise<EmailTrigger> {
        const response = await apiClient.get<ApiResponse<EmailTrigger>>(`${this.baseUrl}/${id}`)
        return response.data
    }

    // 创建触发器
    async createTrigger(data: CreateTriggerRequest): Promise<EmailTrigger> {
        const response = await apiClient.post<ApiResponse<EmailTrigger>>(this.baseUrl, data)
        return response.data
    }

    // 更新触发器
    async updateTrigger(id: number, data: UpdateTriggerRequest): Promise<EmailTrigger> {
        const response = await apiClient.put<ApiResponse<EmailTrigger>>(`${this.baseUrl}/${id}`, data)
        return response.data
    }

    // 删除触发器
    async deleteTrigger(id: number): Promise<void> {
        await apiClient.delete(`${this.baseUrl}/${id}`)
    }

    // 启用触发器
    async enableTrigger(id: number): Promise<EmailTrigger> {
        const response = await apiClient.post<ApiResponse<EmailTrigger>>(`${this.baseUrl}/${id}/enable`)
        return response.data
    }

    // 禁用触发器
    async disableTrigger(id: number): Promise<EmailTrigger> {
        const response = await apiClient.post<ApiResponse<EmailTrigger>>(`${this.baseUrl}/${id}/disable`)
        return response.data
    }

    // 获取触发器执行日志
    async getTriggerLogs(
        triggerId?: number,
        params?: PaginationParams & {
            status?: string
            start_date?: string
            end_date?: string
        }
    ): Promise<PaginatedTriggerLogsResponse> {
        const queryParams = new URLSearchParams()

        if (triggerId) queryParams.append('trigger_id', triggerId.toString())
        if (params?.page) queryParams.append('page', params.page.toString())
        if (params?.limit) queryParams.append('limit', params.limit.toString())
        if (params?.status) queryParams.append('status', params.status)
        if (params?.start_date) queryParams.append('start_date', params.start_date)
        if (params?.end_date) queryParams.append('end_date', params.end_date)

        const url = queryParams.toString() ? `${this.baseUrl}/logs?${queryParams}` : `${this.baseUrl}/logs`
        const response = await apiClient.get<ApiResponse<PaginatedTriggerLogsResponse>>(url)
        return response.data
    }
    
    // 获取单个触发器执行日志
    async getTriggerLog(logId: number): Promise<TriggerExecutionLog> {
        const response = await apiClient.get<ApiResponse<TriggerExecutionLog>>(`${this.baseUrl}/logs/${logId}`)
        return response.data
    }

    // 获取触发器统计信息
    async getTriggerStatistics(
        triggerId: number,
        startDate?: string,
        endDate?: string
    ): Promise<TriggerStatistics> {
        const queryParams = new URLSearchParams()
        if (startDate) queryParams.append('start_date', startDate)
        if (endDate) queryParams.append('end_date', endDate)

        const url = queryParams.toString()
            ? `${this.baseUrl}/${triggerId}/statistics?${queryParams}`
            : `${this.baseUrl}/${triggerId}/statistics`

        const response = await apiClient.get<ApiResponse<TriggerStatistics>>(url)
        return response.data
    }

    // 测试触发器条件
    async testTriggerCondition(condition: any, emailData: any): Promise<{ result: boolean; error?: string }> {
        const response = await apiClient.post<ApiResponse<{ result: boolean; error?: string }>>(
            `${this.baseUrl}/test-condition`,
            { condition, email_data: emailData }
        )
        return response.data
    }

    // 测试触发器动作
    async testTriggerAction(action: any, emailData: any): Promise<{ result: any; error?: string }> {
        const response = await apiClient.post<ApiResponse<{ result: any; error?: string }>>(
            `${this.baseUrl}/test-action`,
            { action, email_data: emailData }
        )
        return response.data
    }
}

export const triggerService = TriggerService.getInstance()
