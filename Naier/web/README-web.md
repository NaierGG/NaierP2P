# Naier Web — UI Architecture

## Tech Stack

- **Framework**: React 18 + TypeScript
- **Build**: Vite 6
- **Styling**: Tailwind CSS 3.4 + CSS custom properties (HSL)
- **UI Components**: shadcn/ui (Radix UI primitives)
- **State**: Redux Toolkit + RTK Query
- **Icons**: lucide-react

## Layout System

```
┌──────────────────────────────────────────────────┐
│                    AppShell                       │
│  ┌────────────┐ ┌──────────────────────────────┐ │
│  │  Sidebar    │ │        ChatPanel             │ │
│  │  (300px)    │ │  ┌────────────────────────┐  │ │
│  │             │ │  │     ChatHeader         │  │ │
│  │  Search     │ │  ├────────────────────────┤  │ │
│  │  ChannelList│ │  │     MessageList         │  │ │
│  │  (scroll)   │ │  │     (flex-1, scroll)    │  │ │
│  │             │ │  ├────────────────────────┤  │ │
│  │  ─────────  │ │  │     Composer           │  │ │
│  │  User/Nav   │ │  └────────────────────────┘  │ │
│  └────────────┘ └──────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

- `h-screen` flex layout (no overflow on body)
- Sidebar: fixed 300px, scrollable channel list via `ScrollArea`
- ChatPanel: `flex-1`, vertically stacked header/messages/composer

## Folder Structure

```
src/
├── components/
│   ├── layout/          # AppShell, Sidebar, ChatPanel
│   ├── chat/            # ChatHeader, ChannelItem, MessageBubble,
│   │                    #   MessageList, Composer, TypingIndicator
│   └── ui/              # shadcn/ui primitives (Button, Input, Card, etc.)
├── features/
│   ├── auth/            # LoginPage, RegisterPage, KeygenFlow
│   ├── channels/        # ChannelList (main app view, composes layout)
│   ├── messages/        # (legacy — replaced by components/chat/)
│   ├── presence/        # (legacy — replaced by components/chat/)
│   └── settings/        # SettingsPage, ProfileSettings, etc.
├── shared/              # hooks, lib, types (unchanged)
├── lib/utils.ts         # cn() helper for Tailwind class merging
└── styles.css           # Tailwind directives + CSS custom properties
```

## Design Tokens (CSS Custom Properties)

All colors use HSL format via `hsl(var(--name))`:

| Token          | Purpose                |
| -------------- | ---------------------- |
| `--background` | Page background        |
| `--foreground` | Primary text           |
| `--card`       | Card/surface bg        |
| `--primary`    | Accent/CTA color       |
| `--muted`      | Subdued surfaces       |
| `--border`     | Borders                |
| `--sidebar`    | Sidebar background     |
| `--bubble`     | Message bubble bg      |
| `--bubble-own` | Own message bubble bg  |

Theme switching: `[data-theme="light"]` overrides all tokens.

## shadcn/ui Components Used

Button, Input, Textarea, Card, ScrollArea, Separator, Tabs, Badge, Dialog, DropdownMenu, Switch

## UX Principles Applied

- **Fitts's Law**: Frequently used controls (send button, navigation) placed at edges/corners
- **Hick's Law**: Settings grouped into tabs to reduce decision complexity
- **Proximity**: Related information clustered (channel name + preview + timestamp)
- **Progressive disclosure**: Key generation uses a step-by-step wizard
- **Auto-growing textarea**: Composer height matches content for reduced cognitive load
- **Relative timestamps**: "방금", "3분", "2시간" instead of absolute dates
- **Connection status**: Always-visible badge in chat header for trust/security perception
