# Admin Dashboard UI Redesign

**Date:** 2025-12-25
**Status:** Design Approved, Ready for Implementation
**Scope:** Dashboard & Login pages (Phase 6 polish)

## Overview

The current admin interface is functional but minimal. This redesign applies professional, warm aesthetics inspired by Figma/Notion while maintaining simplicity for non-technical camp organizers. Goal: first-impression credibility while keeping interfaces fast and uncluttered.

## Visual Direction & Color Palette

**Design Philosophy:** Figma-inspired warm professional - clean, accessible, friendly but polished.

### Colors
- **Primary background:** Cream/off-white (`#FAFAF8`)
- **Card backgrounds:** White (`#FFFFFF`)
- **Primary text:** Dark charcoal (`#2D2D2D`)
- **Secondary text:** Light gray (`#6B7280`)
- **Accent/CTA:** Warm teal (`#2E8B9E`)
- **Published status:** Soft green (`#10B981`)
- **Draft status:** Soft amber (`#F59E0B`)
- **Danger actions:** Soft red (`#EF4444`)
- **Borders:** Subtle gray (`#E5E5E3`)

### Typography
- **Font stack:** System fonts (`-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`)
- **Base unit:** 16px padding/spacing
- **Hierarchy:** Large titles (28px bold), readable body (14-16px), secondary labels (12px gray)

## Login Page

**Purpose:** Entry point for camp organizers - must feel trustworthy and simple.

**Layout:**
- Full viewport with cream background
- Centered white card (max-width: 400px)
- Top: Logo/branding text "StinkyKitty"
- Heading: "Sign In to Your Site" (28px, dark charcoal)
- Helpful subtext: "One account for all your camps" (14px, light gray)

**Form Elements:**
- Email input: clean border (`#E5E5E3`), gray placeholder, 16px padding
- Password input: same styling
- "Sign In" button: teal background, full width, 16px vertical padding, no shadow
- Focus states: teal bottom border highlight, no box-shadow
- Button hover: darker teal (`#1E6F7F`)
- "Forgot password?" link below (if supported): teal color, no underline

**Spacing:** 24px vertical gaps between sections, 16px padding inside card

## Admin Dashboard

**Purpose:** Camp organizers see their pages, create new ones, manage navigation. Get in, edit, get out.

### Header Bar (Sticky Top)
- Background: White with subtle bottom border (`#E5E5E3`)
- Left side: Site name/logo + "Your Site" heading (dark, 18px)
- Right side: User email (14px gray) + Logout button (text link, teal)
- 20px padding top/bottom, 24px padding left/right
- Subtle drop shadow for depth

### Hero Section
- Padding: 40px top/bottom, 24px left/right
- Centered "Create New Page" button (teal, 16px padding, rounded 6px)
- Subtext below: "or manage existing pages below" (14px gray, centered)
- Breathing room emphasizes this as primary action

### Pages List
- Each page is a **card**:
  - White background, subtle border, rounded corners (6px)
  - 16px padding inside
  - 12px bottom margin between cards
  - Hover state: slight lift (box-shadow increase), background very slightly darker

- **Card layout (flex, space-between):**
  - Left: Page title (bold 16px), slug underneath (12px gray)
  - Middle: Status badge (green "Published" or amber "Draft", 12px, rounded 4px)
  - Right: Edit button (teal, 12px padding), Delete button (red, 12px padding, confirm on click)

- **Empty state:** If no pages, show centered message: "No pages yet. Create one to get started." with Create button

### Footer Navigation
- Padding: 24px
- "Navigation Menu" button (outlined teal, not filled)
- "View Public Site" link (teal text)
- Subtle divider above section

## Responsive Behavior

- **Desktop (1024px+):** Full layout as described
- **Tablet (768px-1023px):** Pages cards stack, buttons remain accessible
- **Mobile (< 768px):** Single column, buttons stack vertically on cards if needed, header simplifies

## Implementation Notes

- All colors use CSS variables for easy theming
- Use consistent 6px border-radius for modern feel
- Shadows: use subtle `box-shadow: 0 1px 3px rgba(0,0,0,0.1)` for cards
- Hover states: background tint + shadow increase (no color changes, accessibility first)
- Transitions: 200ms ease for smooth interactions

## Success Criteria

- Dashboard feels professional and trustworthy to non-technical users
- No feature bloat - pages list and create button are obvious
- Consistent aesthetic between login and dashboard
- Ready to potentially integrate rich editor later (design won't conflict)
