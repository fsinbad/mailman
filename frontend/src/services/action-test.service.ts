import { apiClient } from '@/lib/api-client'
import { TriggerActionConfig, ApiResponse } from '@/types'

export class ActionTestService {
  private static instance: ActionTestService
  private baseUrl = '/triggers'

  static getInstance(): ActionTestService {
    if (!ActionTestService.instance) {
      ActionTestService.instance = new ActionTestService()
    }
    return ActionTestService.instance
  }

  // 测试触发器动作
  async testAction(action: TriggerActionConfig, emailData: any): Promise<{
    success: boolean
    result?: any
    error?: string
    executionTime?: number
  }> {
    try {
      const startTime = performance.now()
      
      const response = await apiClient.post<ApiResponse<any>>(
        `${this.baseUrl}/test-action`,
        { action, email_data: emailData }
      )
      
      const endTime = performance.now()
      const executionTime = Math.round(endTime - startTime)
      
      if (response.error) {
        return {
          success: false,
          error: response.error,
          executionTime
        }
      }
      
      return {
        success: true,
        result: response.data,
        executionTime
      }
    } catch (error: any) {
      return {
        success: false,
        error: error.message || '测试动作失败'
      }
    }
  }

  // 获取动作类型列表
  async getActionTypes(): Promise<Array<{
    id: string
    name: string
    description: string
    configSchema: any
  }>> {
    try {
      const response = await apiClient.get<ApiResponse<any>>(`${this.baseUrl}/action-types`)
      return response.data || []
    } catch (error) {
      console.error('获取动作类型失败:', error)
      return []
    }
  }

  // 验证动作配置
  async validateActionConfig(actionType: string, config: any): Promise<{
    valid: boolean
    errors?: Record<string, string>
  }> {
    try {
      const response = await apiClient.post<ApiResponse<any>>(
        `${this.baseUrl}/validate-action-config`,
        { action_type: actionType, config }
      )
      
      return {
        valid: !response.error,
        errors: response.error ? { general: response.error } : undefined
      }
    } catch (error: any) {
      return {
        valid: false,
        errors: { general: error.message || '验证配置失败' }
      }
    }
  }
}

export const actionTestService = ActionTestService.getInstance()