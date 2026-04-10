import axios from 'axios';

export const client = axios.create({
  baseURL: '/api/v1',
});

export interface Device {
  id: string;
  ipv4: string;
  ipv6: string;
  mac: string;
  vendor: string;
  type: string;
  last_seen: string;
  confidence: number;
}

export interface Link {
  id: string;
  a_device: string;
  a_interface: string;
  b_device: string;
  b_interface: string;
  media: string;
  speed_mbps: number;
  source: string;
  confidence: number;
}

export async function fetchDevices() {
  const { data } = await client.get<Device[]>('/devices');
  return data;
}

export async function fetchLinks() {
  const { data } = await client.get<Link[]>('/links');
  return data;
}
