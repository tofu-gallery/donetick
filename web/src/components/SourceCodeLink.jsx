// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream
// Donetick. It is licensed under the AGPL-3.0, like the rest of this app.
//
// AGPL-3.0 section 13: everyone who uses a copy over the network is offered
// that copy's own source code. The server names it in /api/v1/resource as
// source_url (the DT_SOURCE_URL setting; Tofu's image sets it to the exact
// version it was built from). Before the server has answered, or when it
// names none, the link opens Tofu's fork of Donetick.
import { Link } from '@mui/joy'
import { useTranslation } from 'react-i18next'

import { useResource } from '../queries/ResourceQueries'

export const FALLBACK_SOURCE_URL = 'https://github.com/tofu-gallery/donetick'

export const useSourceUrl = () => {
  const { data: resource } = useResource()
  return resource?.source_url || FALLBACK_SOURCE_URL
}

const SourceCodeLink = ({ sx, ...props }) => {
  const { t } = useTranslation('auth')
  const href = useSourceUrl()

  return (
    <Link
      href={href}
      target='_blank'
      rel='noopener'
      color='neutral'
      underline='hover'
      sx={sx}
      {...props}
    >
      {t('sourceCode', { defaultValue: 'Source code' })}
    </Link>
  )
}

export default SourceCodeLink
