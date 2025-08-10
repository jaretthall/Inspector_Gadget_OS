import { useQuery, useMutation } from '@tanstack/react-query'
import api from '@services/api'
import { Box, Button, Paper, Stack, TextField, Typography } from '@mui/material'
import { useState } from 'react'

export default function Gadgets() {
  const { data, refetch } = useQuery({
    queryKey: ['gadgets'],
    queryFn: async () => (await api.get('/api/gadgets')).data,
  })

  const [selected, setSelected] = useState('')
  const [args, setArgs] = useState('')

  const exec = useMutation({
    mutationFn: async () => {
      const name = selected.trim()
      const argList = args.trim() ? args.trim().split(/\s+/) : []
      return (await api.post(`/api/gadgets/${name}/execute`, { gadget_name: name, args: argList })).data
    }
  })

  return (
    <Stack spacing={2}>
      <Paper sx={{ p: 2 }}>
        <Typography variant="h6">Available Gadgets ({data?.count ?? 0})</Typography>
        <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mt: 1 }}>
          {data?.gadgets?.map((g: any) => (
            <Button key={g.name} variant={selected === g.name ? 'contained' : 'outlined'} onClick={() => setSelected(g.name)}>
              {g.name}
            </Button>
          ))}
          <Button onClick={() => refetch()}>Refresh</Button>
        </Box>
      </Paper>
      <Paper sx={{ p: 2 }}>
        <Typography variant="h6">Execute</Typography>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
          <TextField label="Gadget" value={selected} onChange={e => setSelected(e.target.value)} />
          <TextField label="Args" placeholder="space separated" value={args} onChange={e => setArgs(e.target.value)} fullWidth />
          <Button variant="contained" onClick={() => exec.mutate()} disabled={!selected || exec.isPending}>Run</Button>
        </Stack>
        {exec.data && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2">Exit: {exec.data.exit_code} Success: {String(exec.data.success)}</Typography>
            <pre style={{ whiteSpace: 'pre-wrap' }}>{exec.data.output || exec.data.error}</pre>
          </Box>
        )}
      </Paper>
    </Stack>
  )
}


