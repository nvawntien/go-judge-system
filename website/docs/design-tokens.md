# Design Tokens

File này là source of truth cho visual convention của toàn bộ frontend online judge.

Không tự ý sửa file này nếu task không yêu cầu thay đổi visual system hoặc redesign toàn cục.

---

## Purpose

File này định nghĩa:

- color palette
- semantic colors
- typography convention
- spacing convention
- layout convention
- component visual rules
- status/difficulty badge rules
- online judge screen patterns
- visual anti-patterns

Codex và frontend engineer phải đọc file này trước khi làm task liên quan UI/UX, layout, màu sắc, badge, card, table, form hoặc responsive behavior.

---

## Product direction

Sản phẩm là hệ thống online judge / developer platform / education platform.

Định hướng UI đã chốt:

- modern developer SaaS dashboard
- clean
- technical
- focused
- fast scanning
- premium nhưng không màu mè
- giống developer platform hơn là admin dashboard nặng

Ưu tiên UX:

- đọc đề bài nhanh
- tìm bài nhanh
- lọc bài dễ
- submit code rõ ràng
- trạng thái submission dễ scan
- responsive tốt
- contrast tốt
- visual hierarchy rõ
- giảm cognitive load

Không thiết kế kiểu generic AI.

Không dùng quá nhiều gradient, glassmorphism hoặc animation nếu không cần.

---

## Final layout decision

Layout chính đã chốt:

```txt
Unified top navigation layout
White + Violet + Slate
Admin dùng secondary nav ngang nếu cần
Không dùng sidebar trái mặc định
```

Toàn bộ frontend ưu tiên dùng một navigation pattern thống nhất: top navigation bar ngang.

Không dùng sidebar trái làm layout mặc định.

### Global navigation

Top navigation dùng cho cả user-facing pages và admin pages.

Top nav nên gồm:

- Logo
- Problems
- Contests
- Submissions
- Ranking
- Manage/Admin nếu user có quyền admin
- Global search
- Theme toggle nếu có
- Notification nếu có
- User menu/avatar

### Admin navigation

Admin không dùng sidebar trái mặc định.

Nếu admin cần nhiều mục quản lý, dùng secondary navigation ngang bên dưới top nav:

```txt
Overview | Problems | Contests | Users | Submissions | Settings
```

### Mobile navigation

Trên mobile, top navigation có thể collapse thành:

- hamburger menu
- Sheet/Drawer
- command search
- user dropdown

Mobile navigation vẫn phải dựa trên cùng navigation structure, không tạo hệ menu riêng hoàn toàn.

---

## Color palette

Visual direction đã chốt:

```txt
White + Violet + Slate
```

Palette chính:

```txt
Background: #FFFFFF
App bg:     #FAFAFB
Card:       #FFFFFF
Border:     #E5E7EB

Primary:    #6D28D9
Primary 2:  #8B5CF6
Info:       #2563EB
Success:    #16A34A
Warning:    #F59E0B
Error:      #DC2626

Text:       #111827
Muted:      #6B7280
```

### Token meaning

| Token | Hex | Usage |
|---|---:|---|
| Background | `#FFFFFF` | Page background, reading surfaces |
| App bg | `#FAFAFB` | App shell background, subtle page background |
| Card | `#FFFFFF` | Cards, panels, popovers |
| Border | `#E5E7EB` | Card borders, dividers, input borders |
| Primary | `#6D28D9` | Primary buttons, active nav, selected tabs |
| Primary 2 | `#8B5CF6` | Hover states, soft accent, highlights |
| Info | `#2563EB` | Links, info badges, secondary accent |
| Success | `#16A34A` | Accepted, solved, successful actions |
| Warning | `#F59E0B` | Time limit, memory limit, pending warnings |
| Error | `#DC2626` | Wrong answer, runtime error, destructive actions |
| Text | `#111827` | Main readable text |
| Muted | `#6B7280` | Metadata, helper text, secondary labels |

---

## Color usage rules

### Base

Dùng nền trắng hoặc xám rất nhạt cho phần lớn giao diện.

Ưu tiên:

- page background: `#FFFFFF`
- app shell background: `#FAFAFB`
- card background: `#FFFFFF`
- border: `#E5E7EB`
- text chính: `#111827`
- text phụ: `#6B7280`

Nền chính phải sáng, thoáng và dễ đọc code/đề bài.

### Primary violet

Không dùng tím quá nhiều.

Chỉ dùng tím cho:

- active navigation item
- primary button
- selected tab
- focus ring
- link quan trọng
- important highlight
- progress indicator chính

Không dùng tím làm background lớn cho toàn page.

### Blue info

Dùng xanh dương cho:

- info state
- link phụ
- documentation/helper link
- neutral action không phải primary
- running state nếu cần nổi bật hơn muted

### Success green

Dùng xanh lá cho:

- Accepted
- Solved
- success toast
- test passed
- positive stat

### Warning orange

Dùng cam cho:

- Time Limit Exceeded
- Memory Limit Exceeded
- warning toast
- pending review
- contest almost ending

### Error red

Dùng đỏ cho:

- Wrong Answer
- Runtime Error
- Compilation Error
- destructive actions
- validation error
- failed submit

---

## shadcn/ui CSS variable mapping

Nếu project dùng shadcn/ui, ưu tiên map palette vào CSS variables thay vì hardcode màu ở từng component.

Gợi ý mapping:

```css
:root {
  --background: 0 0% 100%;
  --foreground: 221 39% 11%;

  --card: 0 0% 100%;
  --card-foreground: 221 39% 11%;

  --popover: 0 0% 100%;
  --popover-foreground: 221 39% 11%;

  --primary: 262 83% 58%;
  --primary-foreground: 0 0% 100%;

  --secondary: 240 20% 98%;
  --secondary-foreground: 221 39% 11%;

  --muted: 240 20% 98%;
  --muted-foreground: 220 9% 46%;

  --accent: 262 83% 96%;
  --accent-foreground: 262 83% 38%;

  --destructive: 0 72% 51%;
  --destructive-foreground: 0 0% 100%;

  --border: 220 13% 91%;
  --input: 220 13% 91%;
  --ring: 262 83% 58%;
}
```

Notes:

- `--primary` đại diện cho violet.
- `--ring` dùng violet để focus state nhất quán.
- `--destructive` dùng red.
- Không tự đổi theme system nếu project đã có tokens riêng; hãy inspect trước.

---

## Theme system

Frontend nên hỗ trợ cả light theme và dark theme nếu project đã có hoặc task yêu cầu theme toggle.

Không tự tạo theme system mới nếu project chưa có và task không yêu cầu.

Nếu implement theme toggle, ưu tiên dùng approach phù hợp với Next.js + shadcn/ui, ví dụ class-based dark mode.

Theme phải giữ cùng visual language:

```txt
Light: White + Violet + Slate
Dark:  Dark Slate + Violet accent
```

Không được để light theme và dark theme trông như hai sản phẩm khác nhau.

---

## Light theme tokens

Light theme là theme mặc định.

```txt
Background: #FFFFFF
App bg:     #FAFAFB
Card:       #FFFFFF
Border:     #E5E7EB

Primary:    #6D28D9
Primary 2:  #8B5CF6
Info:       #2563EB
Success:    #16A34A
Warning:    #F59E0B
Error:      #DC2626

Text:       #111827
Muted:      #6B7280
```

Light theme usage:

- nền chính sáng, thoáng
- card trắng
- border xám nhạt
- tím chỉ làm accent
- phù hợp cho đọc đề bài dài và coding ban ngày

---

## Dark theme tokens

Dark theme chính thức:

```txt
Background: #020617
App bg:     #0F172A
Card:       #111827
Card 2:     #1F2937
Border:     #334155

Primary:    #8B5CF6
Primary 2:  #A78BFA
Info:       #60A5FA
Success:    #22C55E
Warning:    #F59E0B
Error:      #F87171

Text:       #F9FAFB
Muted:      #9CA3AF
Subtle:     #64748B
```

Dark theme usage:

- background dùng slate rất đậm, không dùng pure black làm nền chính quá nhiều
- app shell dùng `#0F172A`
- card dùng `#111827`
- card elevated hoặc nested surface có thể dùng `#1F2937`
- border dùng `#334155`
- text chính dùng `#F9FAFB`
- muted text dùng `#9CA3AF`
- primary violet sáng hơn light theme để đủ contrast
- status colors phải vẫn rõ trên nền tối

Không dùng neon quá mức.

Không dùng gradient tím đậm làm nền lớn.

---

## shadcn/ui CSS variable mapping

Nếu project dùng shadcn/ui, ưu tiên map palette vào CSS variables thay vì hardcode màu ở từng component.

Light theme mapping:

```css
:root {
  --background: 0 0% 100%;
  --foreground: 221 39% 11%;

  --card: 0 0% 100%;
  --card-foreground: 221 39% 11%;

  --popover: 0 0% 100%;
  --popover-foreground: 221 39% 11%;

  --primary: 262 83% 58%;
  --primary-foreground: 0 0% 100%;

  --secondary: 240 20% 98%;
  --secondary-foreground: 221 39% 11%;

  --muted: 240 20% 98%;
  --muted-foreground: 220 9% 46%;

  --accent: 262 83% 96%;
  --accent-foreground: 262 83% 38%;

  --destructive: 0 72% 51%;
  --destructive-foreground: 0 0% 100%;

  --border: 220 13% 91%;
  --input: 220 13% 91%;
  --ring: 262 83% 58%;
}
```

Dark theme mapping:

```css
.dark {
  --background: 222 47% 5%;
  --foreground: 210 40% 98%;

  --card: 221 39% 11%;
  --card-foreground: 210 40% 98%;

  --popover: 221 39% 11%;
  --popover-foreground: 210 40% 98%;

  --primary: 258 90% 66%;
  --primary-foreground: 0 0% 100%;

  --secondary: 217 33% 17%;
  --secondary-foreground: 210 40% 98%;

  --muted: 217 33% 17%;
  --muted-foreground: 215 20% 65%;

  --accent: 262 83% 20%;
  --accent-foreground: 255 92% 90%;

  --destructive: 0 91% 71%;
  --destructive-foreground: 222 47% 5%;

  --border: 215 25% 27%;
  --input: 215 25% 27%;
  --ring: 258 90% 66%;
}
```

Notes:

- `--primary` đại diện cho violet.
- `--ring` dùng violet để focus state nhất quán.
- `--destructive` dùng red.
- Không tự đổi theme system nếu project đã có tokens riêng; hãy inspect trước.

---

## Dark theme component rules

### Top navbar

Dark mode navbar:

- background: `#0F172A` hoặc token `background/card`
- border-bottom: `#334155`
- active item: violet
- hover: subtle violet/slate
- text chính: `#F9FAFB`
- muted nav item: `#9CA3AF`

### Cards

Dark mode cards:

- card background: `#111827`
- border: `#334155`
- nested card/elevated surface: `#1F2937`
- shadow hạn chế, ưu tiên border

### Tables

Dark mode tables:

- header background subtle slate
- row divider dùng border slate
- row hover dùng slate đậm hơn một chút
- badge màu phải đủ contrast

### Forms

Dark mode forms:

- input background: slate/card
- border: slate border
- focus ring: violet
- placeholder: muted
- error message: red sáng

### Code editor

Code editor có thể dùng theme tối riêng.

Rules:

- dark app + dark editor là default tốt cho coding
- light app + dark editor vẫn được nếu UX tốt
- editor border phải rõ
- editor toolbar phải match theme hiện tại

### Status badges

Status badges trong dark mode phải dùng nền nhẹ và chữ rõ.

Không dùng chỉ màu text mỏng trên nền tối.

Ví dụ semantic direction:

```txt
Accepted: nền xanh lá đậm nhẹ + text xanh lá sáng
Wrong Answer: nền đỏ đậm nhẹ + text đỏ sáng
Time Limit: nền cam đậm nhẹ + text cam sáng
Pending: nền slate/info nhẹ + text muted/info sáng
```

---

## Status colors

Các trạng thái submission phải nhất quán toàn app.

### Accepted

Token:

```txt
Success
```

Label:

```txt
Accepted
```

Usage:

- badge
- result card
- solved state
- positive chart segment

### Pending

Token:

```txt
Muted hoặc Info
```

Label:

```txt
Pending
```

Usage:

- badge
- queue state
- submission đang chờ xử lý

### Judging / Running

Token:

```txt
Info
```

Label:

```txt
Judging
Running
```

Usage:

- badge
- live status
- spinner/progress state nếu có

### Wrong Answer

Token:

```txt
Error
```

Label:

```txt
Wrong Answer
```

### Time Limit Exceeded

Token:

```txt
Warning
```

Label:

```txt
Time Limit Exceeded
```

### Memory Limit Exceeded

Token:

```txt
Warning
```

Label:

```txt
Memory Limit Exceeded
```

### Runtime Error

Token:

```txt
Error
```

Label:

```txt
Runtime Error
```

### Compilation Error

Token:

```txt
Error hoặc Warning
```

Label:

```txt
Compilation Error
```

Default: dùng Error nếu đây là final failed state.

### System Error

Token:

```txt
Error
```

Label:

```txt
System Error
```

---

## Difficulty colors

Difficulty phải nhất quán trên toàn app.

### Easy

Token:

```txt
Success
```

Label:

```txt
Easy
```

### Medium

Token:

```txt
Warning
```

Label:

```txt
Medium
```

### Hard

Token:

```txt
Error
```

Label:

```txt
Hard
```

Không chỉ dựa vào màu. Badge text phải rõ.

---

## Badge conventions

Badges dùng cho:

- difficulty
- submission status
- problem tags
- solved/attempted
- role
- contest state
- visibility state: public/hidden

Rules:

- badge text phải ngắn, rõ
- status badge phải có semantic color
- difficulty badge phải nhất quán
- không dùng quá nhiều badge trong một dòng
- mobile phải không bị wrap xấu

Suggested variants:

```txt
Accepted / Solved: success
Wrong Answer / Runtime Error / Compile Error: destructive
Time Limit / Memory Limit: warning
Pending / Judging: info hoặc secondary
Easy: success
Medium: warning
Hard: destructive
Hidden: secondary
Published: success
```

---

## Typography

Ưu tiên typography dễ đọc và scan nhanh.

Rules:

- Heading rõ hierarchy.
- Body text dễ đọc khi đọc đề bài dài.
- Metadata dùng muted text.
- Code dùng monospace.
- Table text phải dễ scan.
- Không dùng font quá decorative.
- Không dùng quá nhiều font weight trong cùng một màn hình.

Recommended hierarchy:

```txt
Page title:        text-2xl / font-semibold
Section title:     text-lg / font-semibold
Card title:        text-sm hoặc text-base / font-medium
Body:              text-sm hoặc text-base
Metadata:          text-xs hoặc text-sm / muted
Code:              monospace
```

Problem statement:

- ưu tiên `prose` hoặc typography spacing tương đương
- line-height thoáng
- heading rõ
- example block dễ đọc
- constraints không quá dày

---

## Spacing

Spacing phải nhất quán.

Ưu tiên scale theo Tailwind:

```txt
1, 2, 3, 4, 6, 8, 10, 12
```

Không dùng spacing ngẫu nhiên như:

```txt
mt-[17px]
gap-[13px]
p-[19px]
```

trừ khi có lý do rõ ràng.

Layout lớn nên dùng:

```txt
p-4, p-6, p-8
gap-4, gap-6
space-y-4, space-y-6
```

Suggested page spacing:

```txt
Page container: px-4 md:px-6 lg:px-8
Section gap:    gap-6
Card padding:   p-4 hoặc p-6
Table cell:     px-4 py-3
Form gap:       space-y-4
```

---

## Border radius

Ưu tiên radius đồng bộ với shadcn/ui.

Rules:

- card, dialog, popover: rounded-xl hoặc theo token shadcn
- button/input: theo primitive shadcn
- badge: rounded-full hoặc rounded-md tùy component hiện có
- không mix quá nhiều radius style trong cùng màn hình

---

## Shadow and border

Ưu tiên border/subtle elevation thay vì shadow nặng.

Rules:

- card dùng border nhẹ
- shadow chỉ dùng nhẹ cho dropdown/popover/dialog
- không dùng shadow đậm kiểu landing page marketing cho dashboard
- table nên dùng border/divider rõ nhưng nhẹ

---

## Layout convention

### App shell

App shell chính:

- global top navbar
- content area
- optional secondary navigation cho admin
- không dùng sidebar trái mặc định

Desktop:

- top navbar cố định hoặc sticky nếu hữu ích
- content max width tùy màn hình
- problem detail có thể dùng split layout statement/editor
- admin overview dùng card grid/table/chart

Mobile:

- top nav collapse thành Sheet/Drawer
- content không bị horizontal overflow
- action chính dễ bấm
- table có thể chuyển thành card list nếu quá chật

### Page container

Default page container:

```txt
max-w-screen-2xl mx-auto px-4 md:px-6 lg:px-8 py-6
```

Reading-focused page có thể dùng width hẹp hơn cho phần đề bài:

```txt
max-w-4xl
```

Problem solving page có thể dùng split layout rộng hơn:

```txt
grid lg:grid-cols-[minmax(0,1fr)_minmax(420px,0.9fr)]
```

---

## Component conventions

### Top navbar

Top navbar nên có:

- logo rõ
- primary nav items
- active state
- global search
- user menu/avatar
- theme toggle nếu có
- admin/manage item nếu có quyền

Visual rules:

- background trắng hoặc app bg
- border-bottom nhẹ
- active item dùng violet
- hover dùng muted/soft violet
- không quá cao
- không chứa quá nhiều item nhỏ gây rối

### Secondary admin navigation

Dùng cho admin/manage pages khi cần nhiều mục:

```txt
Overview | Problems | Contests | Users | Submissions | Settings
```

Rules:

- nằm dưới global top nav
- dạng horizontal tabs/nav
- active state dùng violet
- không biến thành sidebar nếu chưa được yêu cầu

### Cards

Dùng cho:

- overview KPI
- problem metadata
- submission result
- contest summary
- admin dashboard block

Rules:

- card background trắng
- border nhẹ
- padding nhất quán
- title ngắn
- value nổi bật
- metadata dùng muted text
- không nhồi quá nhiều nội dung vào một card

### Tables

Dùng cho:

- problem list desktop
- submission list desktop
- ranking
- admin data

Rules:

- header rõ
- row hover nhẹ
- status/difficulty dùng badge
- action nằm cuối row
- mobile có thể chuyển thành card list
- không để table gây horizontal overflow trên mobile

### Forms

Form phải có:

- label
- helper text nếu cần
- validation message
- loading state
- disabled state khi submit
- error state rõ ràng

Rules:

- không chỉ dùng placeholder thay label
- input focus ring dùng violet
- primary submit button dùng violet
- destructive action dùng red

### Dialogs

Dùng cho:

- confirm delete
- destructive action
- important confirmation
- small forms nếu phù hợp

Rules:

- title rõ
- description ngắn
- action chính rõ
- destructive action màu error
- keyboard accessible thông qua shadcn primitive

### Empty state

Danh sách rỗng phải có empty state.

Empty state nên nói rõ:

- chưa có gì
- người dùng có thể làm gì tiếp
- action tiếp theo nếu có

Ví dụ:

```txt
No submissions yet.
Submit a solution to see your results here.
```

### Loading state

Dùng Skeleton nếu phù hợp.

Không để màn hình trắng khi loading data.

Loading state phải giữ layout gần giống content thật để tránh layout shift.

### Error state

Error state phải có message rõ.

Nếu có thể, thêm retry action.

Error không nên chỉ hiển thị raw technical message từ backend.

---

## Overview page pattern

Overview page đã chốt theo hướng clean dashboard với top navigation.

Overview page nên gồm:

- global top navbar
- secondary admin nav nếu đang ở Manage/Admin
- page header rõ ràng
- KPI cards
- submission trend chart
- recent submissions table
- problem difficulty distribution
- active contests
- top users/ranking preview
- system status nếu là admin

Overview page không được dùng sidebar trái.

KPI cards nên dùng:

- Total Users
- Total Problems
- Total Submissions
- Accepted Submissions
- Contests

Status/badge màu phải tuân thủ palette đã chốt:

- Accepted / Solved: Success
- Wrong Answer / Runtime Error / Compile Error: Error
- Time Limit / Memory Limit: Warning
- Pending / Running: Info hoặc Muted
- Easy: Success
- Medium: Warning
- Hard: Error

---

## Online judge screen patterns

### Problem list

Cần ưu tiên:

- search rõ
- filter dễ dùng
- difficulty/status badge
- table dễ scan
- mobile không vỡ layout
- empty/loading/error state

Suggested layout:

```txt
Page header
Search + filters
Problem table/card list
Pagination
```

### Problem detail

Cần ưu tiên:

- đề bài dễ đọc
- examples rõ
- constraints rõ
- submit panel dễ thấy
- Monaco Editor dễ dùng
- language selector rõ
- submit button rõ

Suggested layout desktop:

```txt
Top navbar
Problem header
Split layout:
  Left: statement/examples/constraints
  Right: code editor/submit panel
```

Suggested layout mobile:

```txt
Problem statement
Tabs or collapsible submit panel
Code editor fallback/responsive handling
```

### Submit code panel

Cần có:

- language selector
- Monaco Editor
- submit button
- loading state khi submitting
- last submission/result preview nếu có
- error state nếu submit fail

Rules:

- không dùng textarea làm final UI nếu Monaco đã được duyệt
- editor phải có chiều cao hợp lý
- submit button phải dễ thấy
- không làm editor bị quá nhỏ trên desktop

### Submission list

Cần ưu tiên:

- status dễ scan
- problem link rõ
- language rõ
- runtime/memory rõ
- submitted time rõ
- filter nếu có nhiều submission

### Submission detail

Cần có:

- status summary
- problem link
- language
- runtime
- memory
- compile output nếu có
- failed test summary nếu có
- source code viewer/editor read-only nếu cần

### Contest pages

Nếu backend chưa có contest route, chỉ mock khi contract cho phép.

Cần ưu tiên:

- contest list dễ scan
- trạng thái contest rõ: upcoming/running/ended
- thời gian bắt đầu/kết thúc
- participants nếu có
- link vào contest detail

### Auth pages

Cần ưu tiên:

- form đơn giản
- error message dễ hiểu
- loading state khi submit
- link login/register rõ
- không lưu token vào localStorage/sessionStorage

### Admin pages

Admin pages vẫn dùng global top navigation.

Nếu cần nhiều mục quản lý, dùng secondary nav ngang.

Cần ưu tiên:

- table rõ
- filter/search
- bulk action nếu có
- destructive action có confirm
- form tạo/sửa problem rõ
- upload testcase feedback rõ

---

## Code editor visual rules

Monaco Editor là editor mặc định cho submit code.

Visual rules:

- editor container có border rõ
- height đủ lớn
- toolbar có language selector và submit action
- editor không bị chìm giữa quá nhiều card
- theme editor nên phù hợp với app theme
- code font monospace
- mobile cần fallback layout hợp lý

Không dùng textarea làm final submit UI nếu Monaco đã được duyệt.

---

## Accessibility rules

Luôn giữ accessibility cơ bản:

- button có label rõ
- input có label hoặc aria-label
- focus state không bị mất
- dialog/menu/sheet dùng component có keyboard behavior tốt
- không dùng màu sắc làm tín hiệu duy nhất
- text contrast đủ tốt
- icon-only button phải có aria-label hoặc tooltip rõ

Status badge phải có text, không chỉ có màu.

---

## Motion rules

Không lạm dụng animation.

Được dùng:

- hover transition nhẹ
- focus transition nhẹ
- loading spinner/skeleton
- Sheet/Dialog transition có sẵn từ component

Không dùng:

- animation phức tạp không cần thiết
- page transition nặng
- parallax
- excessive glassmorphism
- excessive gradient animation

---

## Visual anti-patterns

Tránh:

- sidebar trái làm layout mặc định
- gradient quá nhiều
- glassmorphism khó đọc
- animation phức tạp không cần thiết
- table quá chật trên mobile
- text contrast thấp
- dùng màu làm tín hiệu duy nhất
- card lồng card quá nhiều
- component quá to và nhiều trách nhiệm
- hardcode màu lung tung
- mỗi page một style khác nhau
- admin layout quá khác user layout
- quá nhiều badge/action trong một row
- quá nhiều màu tím trong cùng màn hình

---

## Change policy

Không tự ý đổi file này.

Chỉ update file này khi:

- user yêu cầu đổi visual direction
- palette chính thức thay đổi
- layout decision thay đổi
- component convention thay đổi
- design system được cập nhật
- dark theme token hoặc theme behavior thay đổi

Nếu task chỉ là implement một page/component, không sửa design tokens.

Nếu cần token mới, phải báo rõ:

- token mới là gì
- dùng ở đâu
- vì sao token hiện có không đủ
- ảnh hưởng tới các màn hình khác không
