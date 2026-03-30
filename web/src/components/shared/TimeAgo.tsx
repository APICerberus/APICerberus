import { useEffect, useState } from "react";
import { formatDistanceToNowStrict } from "date-fns";
import { formatDateTime } from "@/lib/utils";

type TimeAgoProps = {
  value: Date | string | number;
  className?: string;
};

export function TimeAgo({ value, className }: TimeAgoProps) {
  const [text, setText] = useState(() => formatDistanceToNowStrict(new Date(value), { addSuffix: true }));

  useEffect(() => {
    setText(formatDistanceToNowStrict(new Date(value), { addSuffix: true }));
    const timer = window.setInterval(() => {
      setText(formatDistanceToNowStrict(new Date(value), { addSuffix: true }));
    }, 30_000);
    return () => {
      window.clearInterval(timer);
    };
  }, [value]);

  return (
    <time className={className} dateTime={new Date(value).toISOString()} title={formatDateTime(value)}>
      {text}
    </time>
  );
}

