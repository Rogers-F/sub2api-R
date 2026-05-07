/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // ============================================================
        // Xingsuan v2 (品牌主体)
        // ============================================================
        // ink: 暗色 surface(墨黑)
        ink: {
          950: '#08090C',
          900: '#0B0D14',
          800: '#11141C',
          700: '#181C26',
          600: '#222631',
          500: '#2C313D'
        },
        // paper: 亮色 surface(米白,与 Warm Minimalist 同源但更冷一些)
        paper: {
          0:   '#FFFFFF',
          50:  '#FAF9F6',
          100: '#F5F4EE',
          200: '#EFEDE5',
          300: '#E5E2D8'
        },
        // pearl: 暗色文字层(深底上的浅文字)
        pearl: {
          50:  '#F4F5F8',
          100: '#E5E7EE',
          200: '#B8BCC8',
          300: '#8B91A0',
          400: '#6E7385',
          500: '#5A5F6F'
        },
        // dust: 亮色文字层(浅底上的深文字)
        dust: {
          300: '#A8A8A0',
          400: '#8B8B85',
          500: '#5C5C5C',
          600: '#4A4035',
          700: '#403028',
          800: '#2A1F18',
          900: '#1A1308'
        },
        // gold: 跨主题品牌 DNA(光暗都用,深档在 light 上,浅档在 dark 上)
        gold: {
          100: '#F1E4C8',
          200: '#E5D2A4',
          300: '#D4B681',
          400: '#C29F60',
          500: '#A88347',
          600: '#8B6938',
          700: '#6B4F26'
        },
        // aurora: 数据色 - 蓝(light 用 500/600,dark 用 400/500)
        aurora: {
          300: '#B6C5F4',
          400: '#9DB1F0',
          500: '#7B96E8',
          600: '#4862B8',
          700: '#2E47A0'
        },
        // mint: 正向状态(light 用深,dark 用浅)
        mint: {
          400: '#7BD0BC',
          500: '#5AC0A8',
          600: '#2C9985',
          700: '#1A7868'
        },
        // coral: 异常/危险
        coral: {
          400: '#F39A85',
          500: '#E8826B',
          600: '#C66045',
          700: '#9C4530'
        },

        // ============================================================
        // Legacy(过渡兼容,B/C 阶段会陆续替换为上方 token)
        // ============================================================
        // 主色调 - Teal/Cyan 青色系
        primary: {
          50: '#f0fdfa',
          100: '#ccfbf1',
          200: '#99f6e4',
          300: '#5eead4',
          400: '#2dd4bf',
          500: '#14b8a6',
          600: '#0d9488',
          700: '#0f766e',
          800: '#115e59',
          900: '#134e4a',
          950: '#042f2e'
        },
        // 辅助色 - 深蓝灰
        accent: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617'
        },
        // 深色模式背景
        dark: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617'
        },
        // Claude 风格浅色模式专属色板 (Warm Minimalist)
        warm: {
          50: '#FAF9F6',   // 背景主色 - 米色/Paper
          100: '#F5F5F0',  // 背景次色 - 侧边栏/区块
          200: '#E5E5E0',  // 边框色
          300: '#D6D6D0',  // 深边框/禁用态
          400: '#A8A8A0',  // 占位符文字
          500: '#8B8B85',  // 次级图标
          600: '#6B6B65',  // 次级文字 (备用)
          700: '#5C5C5C',  // 文字次色 - 修正后的对比度
          800: '#403028',  // 标题色 - 暖深褐
          900: '#1A1A1A'   // 文字主色 - 深褐黑 (用于 CTA 按钮背景)
        },
        // 强调色 - 橙褐色 (用于图标、链接、装饰)
        clay: {
          400: '#E8886A',  // 浅色 (hover 背景)
          500: '#D97757',  // 强调色
          600: '#B85C3F'   // 强调色 Hover
        }
      },
      fontFamily: {
        // Xingsuan v2:Inter 主导,系统字体 fallback
        sans: [
          'Inter',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        // 展示字体:Playfair Display 配 Noto Serif SC 中文
        display: [
          '"Playfair Display"',
          '"Noto Serif SC"',
          'Georgia',
          'serif'
        ],
        mono: [
          '"JetBrains Mono"',
          'ui-monospace',
          'SFMono-Regular',
          'Menlo',
          'Monaco',
          'Consolas',
          'monospace'
        ]
      },
      boxShadow: {
        glass: '0 8px 32px rgba(0, 0, 0, 0.08)',
        'glass-sm': '0 4px 16px rgba(0, 0, 0, 0.06)',
        glow: '0 0 20px rgba(20, 184, 166, 0.25)',
        'glow-lg': '0 0 40px rgba(20, 184, 166, 0.35)',
        card: '0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.06)',
        'card-hover': '0 10px 40px rgba(0, 0, 0, 0.08)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)',
        // 暖调阴影 - 用于浅色模式 (带棕色调，更自然)
        'warm-sm': '0 2px 8px rgba(60, 50, 40, 0.04)',
        warm: '0 4px 12px rgba(60, 50, 40, 0.08)',
        'warm-lg': '0 8px 24px rgba(60, 50, 40, 0.12)',
        'warm-xl': '0 12px 32px rgba(60, 50, 40, 0.16)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #14b8a6 0%, #0d9488 100%)',
        'gradient-dark': 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(100, 116, 139, 0.07) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(20, 184, 166, 0.04) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(100, 116, 139, 0.05) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'pulse-mint': 'pulseMint 2.4s ease-in-out infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate',
        drift: 'drift 18s linear infinite',
        marquee: 'marquee 30s linear infinite',
        orbit: 'orbit 12s linear infinite'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(20, 184, 166, 0.25)' },
          '100%': { boxShadow: '0 0 30px rgba(20, 184, 166, 0.4)' }
        },
        // Xingsuan v2 动画
        pulseMint: {
          '0%, 100%': { boxShadow: '0 0 0 4px rgba(90,192,168,0.18)' },
          '50%': { boxShadow: '0 0 0 7px rgba(90,192,168,0.05)' }
        },
        drift: {
          '0%': { transform: 'translateY(0)' },
          '100%': { transform: 'translateY(-12px)' }
        },
        marquee: {
          '0%': { transform: 'translateX(0)' },
          '100%': { transform: 'translateX(-50%)' }
        },
        orbit: {
          from: { transform: 'rotate(0deg)' },
          to: { transform: 'rotate(360deg)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
