import { useEffect, useRef } from 'react';
import { useQueries } from '@tanstack/react-query';
import { fetchDevices, fetchLinks } from '../api/client';
import type { Device, Link } from '../api/client';
import { DataSet, Network } from 'vis-network/standalone';

export function TopologyView() {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const networkRef = useRef<Network | null>(null);
  const [devicesQuery, linksQuery] = useQueries({
    queries: [
      { queryKey: ['devices'], queryFn: fetchDevices, refetchInterval: 30000 },
      { queryKey: ['links'], queryFn: fetchLinks, refetchInterval: 30000 },
    ],
  });

  const devices = (devicesQuery.data ?? []) as Device[];
  const links = (linksQuery.data ?? []) as Link[];

  useEffect(() => {
    if (!containerRef.current) return;
    const nodes = new DataSet(devices.map((d) => ({ id: d.id, label: `${d.id}\n${d.ipv4}` })));
    const edges = new DataSet(links.map((l) => ({ id: l.id, from: l.a_device, to: l.b_device, label: l.media })));
    if (!networkRef.current) {
      networkRef.current = new Network(containerRef.current, { nodes, edges }, { physics: true });
    } else {
      networkRef.current.setData({ nodes, edges });
    }
  }, [devices, links]);

  return (
    <div>
      <h3>拓扑图</h3>
      <div ref={containerRef} style={{ height: 400, border: '1px solid #ddd' }} />
    </div>
  );
}
