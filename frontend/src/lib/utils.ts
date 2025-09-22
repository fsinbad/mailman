import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(date: string | Date): string {
  const d = new Date(date);
  return d.toLocaleString();
}

export function truncate(text: string, length: number): string {
  if (text.length <= length) {
    return text;
  }
  return text.substring(0, length) + "...";
}
