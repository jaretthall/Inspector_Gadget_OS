import { useState } from 'react'
import { Box, Button, Container, Paper, TextField, Typography, Alert } from '@mui/material'
import { useNavigate } from 'react-router-dom'
import api from '@services/api'
import useAuthStore from '@store/auth'

export default function Login() {
  const navigate = useNavigate()
  const { setAuth } = useAuthStore()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    try {
      const res = await api.post('/api/auth/login', { username, password })
      const { token, roles } = res.data
      setAuth({ token, user: username, roles })
      navigate('/')
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Login failed')
    }
  }

  return (
    <Container maxWidth="sm" sx={{ mt: 10 }}>
      <Paper sx={{ p: 4 }}>
        <Typography variant="h5" gutterBottom>Sign in</Typography>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        <Box component="form" onSubmit={onSubmit}>
          <TextField fullWidth label="Username" margin="normal" value={username} onChange={e => setUsername(e.target.value)} />
          <TextField fullWidth label="Password" type="password" margin="normal" value={password} onChange={e => setPassword(e.target.value)} />
          <Button type="submit" variant="contained" fullWidth sx={{ mt: 2 }}>Login</Button>
        </Box>
      </Paper>
    </Container>
  )
}


