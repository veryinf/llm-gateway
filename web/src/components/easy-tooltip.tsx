import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';

type EasyTooltipProps = React.ComponentProps<typeof TooltipContent> & {
  tooltip?: string | React.ReactNode;
};

export function EasyTooltip(props: EasyTooltipProps) {
  const { tooltip, children, ...restProps } = props;
  return (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent {...restProps}>
        <p>{tooltip}</p>
      </TooltipContent>
    </Tooltip>
  );
}
