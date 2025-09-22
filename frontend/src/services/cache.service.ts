/**
 * 客户端缓存服务
 * 提供内存缓存和本地存储缓存功能
 */

interface CacheOptions {
  /** 缓存过期时间（毫秒） */
  ttl?: number;
  /** 是否使用本地存储 */
  useLocalStorage?: boolean;
  /** 缓存命名空间 */
  namespace?: string;
}

interface CacheItem<T> {
  value: T;
  expiry: number | null;
}

class CacheService {
  private memoryCache: Map<string, CacheItem<any>> = new Map();
  private defaultOptions: CacheOptions = {
    ttl: 5 * 60 * 1000, // 默认5分钟
    useLocalStorage: false,
    namespace: 'app-cache'
  };

  /**
   * 设置缓存
   * @param key 缓存键
   * @param value 缓存值
   * @param options 缓存选项
   */
  set<T>(key: string, value: T, options?: CacheOptions): void {
    const opts = { ...this.defaultOptions, ...options };
    const cacheKey = this.getCacheKey(key, opts.namespace);
    const expiry = opts.ttl ? Date.now() + opts.ttl : null;
    
    // 内存缓存
    this.memoryCache.set(cacheKey, { value, expiry });
    
    // 本地存储缓存
    if (opts.useLocalStorage && typeof window !== 'undefined') {
      try {
        localStorage.setItem(
          cacheKey,
          JSON.stringify({ value, expiry })
        );
      } catch (error) {
        console.error('缓存到本地存储失败:', error);
      }
    }
  }

  /**
   * 获取缓存
   * @param key 缓存键
   * @param options 缓存选项
   * @returns 缓存值或undefined
   */
  get<T>(key: string, options?: CacheOptions): T | undefined {
    const opts = { ...this.defaultOptions, ...options };
    const cacheKey = this.getCacheKey(key, opts.namespace);
    
    // 先尝试从内存缓存获取
    const memoryItem = this.memoryCache.get(cacheKey);
    if (memoryItem) {
      if (memoryItem.expiry === null || memoryItem.expiry > Date.now()) {
        return memoryItem.value as T;
      } else {
        // 过期了，删除缓存
        this.delete(key, opts);
      }
    }
    
    // 如果内存缓存没有，尝试从本地存储获取
    if (opts.useLocalStorage && typeof window !== 'undefined') {
      try {
        const item = localStorage.getItem(cacheKey);
        if (item) {
          const parsed = JSON.parse(item) as CacheItem<T>;
          if (parsed.expiry === null || parsed.expiry > Date.now()) {
            // 同步到内存缓存
            this.memoryCache.set(cacheKey, parsed);
            return parsed.value;
          } else {
            // 过期了，删除缓存
            this.delete(key, opts);
          }
        }
      } catch (error) {
        console.error('从本地存储获取缓存失败:', error);
      }
    }
    
    return undefined;
  }

  /**
   * 删除缓存
   * @param key 缓存键
   * @param options 缓存选项
   */
  delete(key: string, options?: CacheOptions): void {
    const opts = { ...this.defaultOptions, ...options };
    const cacheKey = this.getCacheKey(key, opts.namespace);
    
    // 删除内存缓存
    this.memoryCache.delete(cacheKey);
    
    // 删除本地存储缓存
    if (opts.useLocalStorage && typeof window !== 'undefined') {
      try {
        localStorage.removeItem(cacheKey);
      } catch (error) {
        console.error('从本地存储删除缓存失败:', error);
      }
    }
  }

  /**
   * 清除指定命名空间下的所有缓存
   * @param namespace 命名空间
   */
  clearNamespace(namespace?: string): void {
    const ns = namespace || this.defaultOptions.namespace;
    
    // 清除内存缓存
    for (const key of this.memoryCache.keys()) {
      if (key.startsWith(`${ns}:`)) {
        this.memoryCache.delete(key);
      }
    }
    
    // 清除本地存储缓存
    if (typeof window !== 'undefined') {
      try {
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          if (key && key.startsWith(`${ns}:`)) {
            localStorage.removeItem(key);
          }
        }
      } catch (error) {
        console.error('清除本地存储缓存失败:', error);
      }
    }
  }

  /**
   * 清除所有缓存
   */
  clear(): void {
    // 清除内存缓存
    this.memoryCache.clear();
    
    // 清除本地存储缓存
    if (typeof window !== 'undefined') {
      try {
        localStorage.clear();
      } catch (error) {
        console.error('清除本地存储缓存失败:', error);
      }
    }
  }

  /**
   * 获取完整的缓存键
   * @param key 原始键
   * @param namespace 命名空间
   * @returns 完整的缓存键
   */
  private getCacheKey(key: string, namespace?: string): string {
    const ns = namespace || this.defaultOptions.namespace;
    return `${ns}:${key}`;
  }
}

// 导出单例
export const cacheService = new CacheService();