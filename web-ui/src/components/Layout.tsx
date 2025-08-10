import { AppBar, Box, Button, Container, CssBaseline, Toolbar, Typography, Stack } from '@mui/material'
import { Link as RouterLink, Outlet, useNavigate } from 'react-router-dom'
import Footer from './Footer'
import useAuthStore from '@store/auth'

export default function Layout() {
  const navigate = useNavigate()
  const { user, roles, logout } = useAuthStore()

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <CssBaseline />
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" sx={{ flexGrow: 1 }}>Inspector Gadget OS</Typography>
          <Stack direction="row" spacing={2}>
            <Button color="inherit" component={RouterLink} to="/">Dashboard</Button>
            <Button color="inherit" component={RouterLink} to="/gadgets">Gadgets</Button>
            <Button color="inherit" component={RouterLink} to="/files">Files</Button>
            {roles.includes('admin') && (
              <Button color="inherit" component={RouterLink} to="/admin/users">Admin</Button>
            )}
            <Button color="inherit" component={RouterLink} to="/settings">Settings</Button>
            <Typography variant="body2">{user}</Typography>
            <Button color="inherit" onClick={() => { logout(); navigate('/login') }}>Logout</Button>
          </Stack>
        </Toolbar>
      </AppBar>
      <Container sx={{ flex: 1, py: 3 }}>
        <Outlet />
      </Container>
      <Footer />
    </Box>
  )
}


