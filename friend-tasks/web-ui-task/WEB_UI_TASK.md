# Inspector Gadget OS - Web UI Task

## Task Overview
Create a modern React-based web dashboard for Inspector Gadget OS that provides a comprehensive interface for managing the system, gadgets, and security features.

## Objectives
Build a complete web-based interface that allows users to:
1. Authenticate with the system (JWT-based)
2. View and manage gadgets through a user-friendly interface
3. Monitor system status and health
4. Manage user roles and permissions (admin interface)
5. Execute gadgets with real-time output display

## Technical Requirements

### Frontend Stack
- **Framework**: React 18+ with TypeScript
- **UI Library**: Material-UI (MUI) or Chakra UI for professional appearance
- **State Management**: React Query for server state + Zustand for client state
- **Routing**: React Router v6
- **HTTP Client**: Axios with interceptors for JWT handling
- **Real-time**: WebSocket or Server-Sent Events for live updates
- **Build Tool**: Vite for fast development

### Key Features to Implement

#### 1. Authentication System
- Login page with username/password
- JWT token management (automatic refresh)
- Role-based navigation and feature access
- Logout functionality

#### 2. Dashboard Overview
- System health status cards
- Active gadgets summary
- Recent activity log
- Quick action buttons

#### 3. Gadget Management Interface
- **Gadget Browser**: Grid/list view of available gadgets
- **Gadget Executor**: Form-based interface for running gadgets with parameters
- **Output Viewer**: Real-time streaming of gadget execution results
- **History**: Past executions with status and results

#### 4. User & Role Management (Admin Only)
- User management table with role assignments
- Permission management interface
- RBAC policy visualization
- System statistics dashboard

#### 5. File System Explorer
- Tree view of allowed directories
- File operations (view, edit simple text files)
- Upload/download capabilities (within security constraints)

#### 6. Settings & Configuration
- MCP server management
- System configuration options
- User preferences

### API Integration Points
Your web UI should integrate with these existing API endpoints:

```
Authentication:
POST /api/auth/login
POST /api/auth/refresh

Health & Status:
GET /health

Gadgets:
GET /api/gadgets
GET /api/gadgets/:name/info
POST /api/gadgets/:name/execute

RBAC Management:
GET /api/rbac/me
GET /api/rbac/users (admin)
GET /api/rbac/roles (admin)
POST /api/rbac/users/:username/roles (admin)

File System:
GET /api/fs/list?path=...
GET /api/fs/read?path=...
POST /api/fs/write

MCP:
GET /api/mcp/servers
POST /api/mcp/servers/:name/connect
GET /api/mcp/resources
```

## Development Guidelines

### Project Structure
```
web-ui/
├── public/
├── src/
│   ├── components/          # Reusable UI components
│   ├── pages/              # Route components
│   ├── hooks/              # Custom React hooks
│   ├── services/           # API integration
│   ├── types/              # TypeScript definitions
│   ├── store/              # State management
│   ├── utils/              # Helper functions
│   └── App.tsx
├── package.json
└── vite.config.ts
```

### Security Considerations
1. **JWT Storage**: Use httpOnly cookies or secure localStorage with proper cleanup
2. **CSRF Protection**: Include CSRF tokens where needed
3. **Input Validation**: Validate all user inputs client-side and expect server validation
4. **Error Handling**: Never expose sensitive information in error messages
5. **Role-Based Access**: Hide/disable features based on user roles

### User Experience Focus
1. **Responsive Design**: Works on desktop, tablet, and mobile
2. **Loading States**: Show spinners/skeletons during API calls
3. **Error Handling**: User-friendly error messages with retry options
4. **Accessibility**: WCAG 2.1 AA compliance
5. **Performance**: Code splitting and lazy loading for large components

## Deliverables

### Phase 1: Core Infrastructure (Week 1)
- [x] Project setup with Vite + React + TypeScript
- [x] Authentication system with JWT handling
- [x] Basic routing structure
- [x] API service layer with error handling
- [x] Basic layout with navigation

### Phase 2: Main Features (Week 2)
- [x] Dashboard with system overview
- [x] Gadget browser and executor
- [x] Real-time output display
- [x] Basic user management (admin)

### Phase 3: Advanced Features (Week 3)
- [x] File system explorer
- [x] Advanced RBAC management
- [x] MCP server management
- [x] Settings and configuration

### Phase 4: Polish & Testing (Week 4)
- [x] Responsive design implementation
- [x] Error handling and edge cases
- [x] Performance optimization
- [x] Basic testing setup
- [x] Documentation

## Integration with Main System

Your web UI will be served by the existing integrated server. Place your built files in:
```
web-ui/dist/ -> should be accessible via the server's static file handler
```

The integrated server will need a static file handler added for serving your React app.

## Testing Strategy
1. **Unit Tests**: React components with React Testing Library
2. **Integration Tests**: API integration with MSW (Mock Service Worker)
3. **E2E Tests**: Key user flows with Playwright or Cypress
4. **Accessibility Testing**: axe-core integration

## Success Criteria
- [ ] Users can authenticate and access role-appropriate features
- [ ] Gadgets can be discovered, configured, and executed through the UI
- [ ] Real-time gadget output is displayed clearly
- [ ] Admin users can manage system users and permissions
- [ ] Interface is responsive and accessible
- [ ] No security vulnerabilities in JWT handling or API calls

## Getting Started
1. Review the existing API endpoints by testing with curl
2. Set up the React project in `web-ui/` folder
3. Implement authentication first (it's required for all other endpoints)
4. Build incrementally, testing each feature with the running integrated server

## Questions & Support
- Test all endpoints with the running integrated server first
- Review the existing codebase for API response formats
- Focus on user experience - make it feel like a professional system administration tool

Good luck! This UI will be the main interface users interact with for Inspector Gadget OS. 🤖✨