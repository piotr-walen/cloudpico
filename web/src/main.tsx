import './index.css'
import "@picocss/pico/css/pico.min.css";

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import Root from './routes/root/root.tsx'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
const queryClient = new QueryClient()

import { BrowserRouter, Route, Routes } from "react-router";
import Layout from './layout.tsx'
import History from './routes/history/history.tsx'



createRoot(document.getElementById('root')!).render(
  <StrictMode>
  <BrowserRouter>
    <QueryClientProvider client={queryClient}>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Root />} />
          <Route path="history" element={<History />} />
        </Route>
      </Routes>
    </QueryClientProvider>
    </BrowserRouter>
  </StrictMode>,
)
