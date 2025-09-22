import { EmailTrigger, PaginatedResponse, PaginationParams } from '@/types';
import { cacheService } from './cache.service';

/**
 * 优化的触发器服务
 * 使用缓存提高性能
 */
class OptimizedTriggerService {
  private apiBaseUrl = '/api/triggers';
  private cacheNamespace = 'triggers';
  private cacheTTL = 5 * 60 * 1000; // 5分钟缓存

  /**
   * 获取触发器列表
   * @param params 分页参数
   * @returns 分页触发器列表
   */
  async getTriggers(params?: PaginationParams): Promise<PaginatedResponse<EmailTrigger>> {
    const queryParams = new URLSearchParams();
    if (params) {
      if (params.page) queryParams.append('page', params.page.toString());
      if (params.limit) queryParams.append('limit', params.limit.toString());
      if (params.search) queryParams.append('search', params.search);
      if (params.status) queryParams.append('status', params.status);
    }
    
    const url = `${this.apiBaseUrl}?${queryParams.toString()}`;
    const cacheKey = `list:${queryParams.toString()}`;
    
    // 尝试从缓存获取
    const cached = cacheService.get<PaginatedResponse<EmailTrigger>>(cacheKey, {
      namespace: this.cacheNamespace,
      ttl: this.cacheTTL
    });
    
    if (cached) {
      return cached;
    }
    
    // 缓存未命中，从API获取
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`获取触发器列表失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // 缓存结果
    cacheService.set(cacheKey, data, {
      namespace: this.cacheNamespace,
      ttl: this.cacheTTL
    });
    
    return data;
  }

  /**
   * 获取单个触发器
   * @param id 触发器ID
   * @returns 触发器详情
   */
  async getTrigger(id: number): Promise<EmailTrigger> {
    const cacheKey = `detail:${id}`;
    
    // 尝试从缓存获取
    const cached = cacheService.get<EmailTrigger>(cacheKey, {
      namespace: this.cacheNamespace,
      ttl: this.cacheTTL
    });
    
    if (cached) {
      return cached;
    }
    
    // 缓存未命中，从API获取
    const response = await fetch(`${this.apiBaseUrl}/${id}`);
    if (!response.ok) {
      throw new Error(`获取触发器详情失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // 缓存结果
    cacheService.set(cacheKey, data, {
      namespace: this.cacheNamespace,
      ttl: this.cacheTTL
    });
    
    return data;
  }

  /**
   * 创建触发器
   * @param trigger 触发器数据
   * @returns 创建的触发器
   */
  async createTrigger(trigger: Partial<EmailTrigger>): Promise<EmailTrigger> {
    const response = await fetch(this.apiBaseUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(trigger)
    });
    
    if (!response.ok) {
      throw new Error(`创建触发器失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // 清除列表缓存
    this.invalidateListCache();
    
    return data;
  }

  /**
   * 更新触发器
   * @param id 触发器ID
   * @param trigger 触发器数据
   * @returns 更新后的触发器
   */
  async updateTrigger(id: number, trigger: Partial<EmailTrigger>): Promise<EmailTrigger> {
    const response = await fetch(`${this.apiBaseUrl}/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(trigger)
    });
    
    if (!response.ok) {
      throw new Error(`更新触发器失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // 更新缓存
    cacheService.set(`detail:${id}`, data, {
      namespace: this.cacheNamespace,
      ttl: this.cacheTTL
    });
    
    // 清除列表缓存
    this.invalidateListCache();
    
    return data;
  }

  /**
   * 删除触发器
   * @param id 触发器ID
   */
  async deleteTrigger(id: number): Promise<void> {
    const response = await fetch(`${this.apiBaseUrl}/${id}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      throw new Error(`删除触发器失败: ${response.statusText}`);
    }
    
    // 删除缓存
    cacheService.delete(`detail:${id}`, {
      namespace: this.cacheNamespace
    });
    
    // 清除列表缓存
    this.invalidateListCache();
  }

  /**
   * 启用触发器
   * @param id 触发器ID
   * @returns 更新后的触发器
   */
  async enableTrigger(id: number): Promise<EmailTrigger> {
    const response = await fetch(`${this.apiBaseUrl}/${id}/enable`, {
      method: 'POST'
    });
    
    if (!response.ok) {
      throw new Error(`启用触发器失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // 更新缓存
    cacheService.set(`detail:${id}`, data, {
      namespace: this.cacheNamespace,
      ttl: this.cacheTTL
    });
    
    // 清除列表缓存
    this.invalidateListCache();
    
    return data;
  }

  /**
   * 禁用触发器
   * @param id 触发器ID
   * @returns 更新后的触发器
   */
  async disableTrigger(id: number): Promise<EmailTrigger> {
    const response = await fetch(`${this.apiBaseUrl}/${id}/disable`, {
      method: 'POST'
    });
    
    if (!response.ok) {
      throw new Error(`禁用触发器失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // 更新缓存
    cacheService.set(`detail:${id}`, data, {
      namespace: this.cacheNamespace,
      ttl: this.cacheTTL
    });
    
    // 清除列表缓存
    this.invalidateListCache();
    
    return data;
  }

  /**
   * 测试触发器条件
   * @param expression 条件表达式
   * @param testData 测试数据
   * @returns 测试结果
   */
  async testTriggerCondition(expression: any, testData: any): Promise<any> {
    const response = await fetch(`${this.apiBaseUrl}/test-condition`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        expression,
        testData
      })
    });
    
    if (!response.ok) {
      throw new Error(`测试触发器条件失败: ${response.statusText}`);
    }
    
    return await response.json();
  }

  /**
   * 测试触发器动作
   * @param action 动作配置
   * @param testData 测试数据
   * @returns 测试结果
   */
  async testTriggerAction(action: any, testData: any): Promise<any> {
    const response = await fetch(`${this.apiBaseUrl}/test-action`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        action,
        testData
      })
    });
    
    if (!response.ok) {
      throw new Error(`测试触发器动作失败: ${response.statusText}`);
    }
    
    return await response.json();
  }

  /**
   * 获取触发器执行日志
   * @param params 查询参数
   * @returns 分页日志列表
   */
  async getTriggerLogs(params?: any): Promise<PaginatedResponse<any>> {
    const queryParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          queryParams.append(key, String(value));
        }
      });
    }
    
    const url = `${this.apiBaseUrl}/logs?${queryParams.toString()}`;
    const cacheKey = `logs:${queryParams.toString()}`;
    
    // 尝试从缓存获取
    const cached = cacheService.get<PaginatedResponse<any>>(cacheKey, {
      namespace: this.cacheNamespace,
      ttl: 60 * 1000 // 日志缓存时间较短，1分钟
    });
    
    if (cached) {
      return cached;
    }
    
    // 缓存未命中，从API获取
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`获取触发器日志失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // 缓存结果
    cacheService.set(cacheKey, data, {
      namespace: this.cacheNamespace,
      ttl: 60 * 1000 // 日志缓存时间较短，1分钟
    });
    
    return data;
  }

  /**
   * 清除列表缓存
   */
  private invalidateListCache(): void {
    // 清除所有以list:开头的缓存
    cacheService.clearNamespace(this.cacheNamespace);
  }
}

// 导出单例
export const optimizedTriggerService = new OptimizedTriggerService();