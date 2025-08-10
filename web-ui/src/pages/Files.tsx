import { useQuery, useMutation } from '@tanstack/react-query'
import api from '@services/api'
import { Box, Button, Paper, Stack, TextField, Typography } from '@mui/material'
import { useState } from 'react'

export default function Files() {
  const [path, setPath] = useState('/tmp')
  const [contentPath, setContentPath] = useState('')
  const [content, setContent] = useState('')

  const list = useQuery({
    queryKey: ['fs-list', path],
    queryFn: async () => (await api.get('/api/fs/list', { params: { path } })).data,
  })

  const read = useMutation({
    mutationFn: async (p: string) => (await api.get('/api/fs/read', { params: { path: p } })).data,
    onSuccess: (data) => { setContentPath(data.path); setContent(data.content) }
  })

  const write = useMutation({
    mutationFn: async () => (await api.post('/api/fs/write', { path: contentPath, content })).data,
  })

  return (
    <Stack spacing={2}>
      <Paper sx={{ p: 2 }}>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
          <TextField label="Directory" value={path} onChange={e => setPath(e.target.value)} fullWidth />
          <Button onClick={() => list.refetch()}>List</Button>
        </Stack>
        <Box sx={{ mt: 2, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
          <Box>
            <Typography variant="subtitle2">Files ({list.data?.count ?? 0})</Typography>
            <ul>
              {list.data?.files?.map((f: any) => (
                <li key={f.name}>
                  <Button onClick={() => read.mutate(`${path}/${f.name}`)}>{f.is_dir ? '[DIR]' : '[FILE]'} {f.name}</Button>
                </li>
              ))}
            </ul>
          </Box>
          <Box>
            <Typography variant="subtitle2">Edit: {contentPath}</Typography>
            <TextField value={contentPath} onChange={e => setContentPath(e.target.value)} label="Path" fullWidth sx={{ mb: 1 }} />
            <TextField value={content} onChange={e => setContent(e.target.value)} label="Content" fullWidth multiline minRows={10} />
            <Button sx={{ mt: 1 }} variant="contained" onClick={() => write.mutate()} disabled={!contentPath}>Save</Button>
          </Box>
        </Box>
      </Paper>
    </Stack>
  )
}


