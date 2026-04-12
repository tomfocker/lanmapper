import { useEffect, useMemo, useRef } from 'react';
import { useQueries } from '@tanstack/react-query';
import { fetchDevices, fetchLinks } from '../api/client';
import type { Device, Link } from '../api/client';
import { DataSet, Network } from 'vis-network/standalone';

const NODE_COLORS: Record<string, string> = {
  router: '#fecdd3',
  switch: '#bfdbfe',
  endpoint: '#e2e8f0',
};

const LINK_STYLES: Record<string, { dashes: boolean | number[]; color: string }> = {
  lldp: { dashes: false, color: '#2563eb' },
  bridge: { dashes: [6, 4], color: '#10b981' },
  gateway: { dashes: [2, 6], color: '#f97316' },
  default: { dashes: false, color: '#94a3b8' },
};

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
    const nodes = new DataSet(
      devices.map((device) => {
        const type = normalizeType(device.type);
        return {
          id: device.id,
          label: formatNodeLabel(device),
          title: buildTooltip(device),
          shape: 'box',
          font: { multi: true },
          color: {
            background: NODE_COLORS[type] || NODE_COLORS.endpoint,
            border: '#334155',
          },
        };
      }),
    );
    const edges = new DataSet(
      links.map((link) => {
        const style = LINK_STYLES[link.kind || 'default'] || LINK_STYLES.default;
        return {
          id: link.id,
          from: link.a_device,
          to: link.b_device,
          label: link.media || link.kind,
          dashes: style.dashes,
          color: { color: style.color },
          title: `${link.kind || 'link'} · 置信度 ${Math.round(link.confidence * 100)}%`,
        };
      }),
    );
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
      <div className="topology-legend">
        {Object.entries(LINK_STYLES).map(([kind, style]) => {
          if (kind === 'default') return null;
          return (
            <span key={kind} className="legend-item">
              <span
                className="legend-line"
                style={{
                  borderBottomStyle: style.dashes ? 'dashed' : 'solid',
                  borderBottomColor: style.color,
                }}
              />
              {kind.toUpperCase()} 链路
            </span>
          );
        })}
      </div>
    </div>
  );
}

function formatNodeLabel(device: Device) {
  const name = device.hostname || device.vendor || device.id;
  const type = device.type || 'Endpoint';
  const ip = device.ipv4 || device.id;
  return `${name}\n(${type})\n${ip}`;
}

function normalizeType(type?: string) {
  return type ? type.toLowerCase() : 'endpoint';
}

function buildTooltip(device: Device) {
  const parts = [
    device.hostname,
    device.vendor,
    device.ipv4,
    device.mac,
    device.type,
  ].filter(Boolean);
  return parts.join(' · ');
}
