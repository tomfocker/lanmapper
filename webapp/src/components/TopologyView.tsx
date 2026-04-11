import { useEffect, useMemo, useRef } from 'react';
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

  const devices = useMemo<Device[]>(() => (Array.isArray(devicesQuery.data) ? devicesQuery.data : []), [devicesQuery.data]);
  const links = useMemo<Link[]>(() => (Array.isArray(linksQuery.data) ? linksQuery.data : []), [linksQuery.data]);
  const isLoading = devicesQuery.isLoading || linksQuery.isLoading;
  const isError = devicesQuery.isError || linksQuery.isError;

  useEffect(() => {
    if (!containerRef.current) return;
    if (!devices.length) {
      if (networkRef.current) {
        networkRef.current.destroy();
        networkRef.current = null;
      }
      return;
    }
    const nodes = new DataSet(devices.map((d) => ({ id: d.id, label: `${d.id}\n${d.ipv4}` })));
    const edges = new DataSet(links.map((l) => ({ id: l.id, from: l.a_device, to: l.b_device, label: l.media })));
    if (!networkRef.current) {
      networkRef.current = new Network(containerRef.current, { nodes, edges }, { physics: true });
    } else {
      networkRef.current.setData({ nodes, edges });
    }
  }, [devices, links]);

  if (isLoading) {
    return (
      <div>
        <h3>拓扑图</h3>
        <div className="topology state">正在扫描拓扑…</div>
      </div>
    );
  }

  if (isError) {
    return (
      <div>
        <h3>拓扑图</h3>
        <div className="topology state error">获取拓扑数据失败</div>
      </div>
    );
  }

  if (!devices.length) {
    return (
      <div>
        <h3>拓扑图</h3>
        <div className="topology empty">尚未发现连接的设备</div>
      </div>
    );
  }

  return (
    <div>
      <h3>拓扑图</h3>
      <div ref={containerRef} className="topology canvas" />
    </div>
  );
}
