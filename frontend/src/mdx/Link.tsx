import { Link } from 'react-router-dom'

/** Lesson links: in-app paths (starting with "/") use the router; everything else is a normal link. */
export default function MdxLink({ href = '', children, ...rest }: React.ComponentProps<'a'>) {
  if (href.startsWith('/')) {
    return (
      <Link to={href} className="font-medium underline decoration-sky-400 underline-offset-2">
        {children}
      </Link>
    )
  }
  return (
    <a href={href} {...rest}>
      {children}
    </a>
  )
}
