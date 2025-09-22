import { useEffect, useRef } from 'react';
import { performanceMonitor } from '@/utils/performance-monitor';

/**
 * 使用性能监控的React Hook
 * @param componentName 组件名称
 * @returns 性能监控工具函数
 */
export function usePerformance(componentName: string) {
  const renderCountRef = useRef(0);
  const renderStartTimeRef = useRef(0);
  
  // 监控组件渲染
  useEffect(() => {
    renderCountRef.current++;
    const duration = performance.now() - renderStartTimeRef.current;
    
    // 记录渲染时间
    if (renderStartTimeRef.current > 0) {
      console.debug(`[Performance] ${componentName} rendered in ${duration.toFixed(2)}ms (count: ${renderCountRef.current})`);
    }
    
    // 组件挂载时记录
    return () => {
      const mountDuration = performance.now() - renderStartTimeRef.current;
      console.debug(`[Performance] ${componentName} unmounted after ${mountDuration.toFixed(2)}ms`);
    };
  });
  
  // 开始新的渲染计时
  renderStartTimeRef.current = performance.now();
  
  /**
   * 测量函数执行时间
   * @param operation 操作名称
   * @param fn 要测量的函数
   * @returns 函数执行结果
   */
  const measure = <T>(operation: string, fn: () => T): T => {
    return performanceMonitor.measure(componentName, operation, fn);
  };
  
  /**
   * 测量异步函数执行时间
   * @param operation 操作名称
   * @param fn 要测量的异步函数
   * @returns Promise<函数执行结果>
   */
  const measureAsync = <T>(operation: string, fn: () => Promise<T>): Promise<T> => {
    return performanceMonitor.measureAsync(componentName, operation, fn);
  };
  
  /**
   * 创建一个被测量的回调函数
   * @param operation 操作名称
   * @param callback 回调函数
   * @returns 被测量的回调函数
   */
  const measuredCallback = <T extends any[]>(
    operation: string,
    callback: (...args: T) => void
  ): ((...args: T) => void) => {
    return (...args: T) => {
      performanceMonitor.measure(componentName, operation, () => {
        callback(...args);
      });
    };
  };
  
  return {
    measure,
    measureAsync,
    measuredCallback,
    renderCount: renderCountRef.current
  };
}