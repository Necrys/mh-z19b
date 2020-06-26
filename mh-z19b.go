package mhz19b

import(
  "github.com/Necrys/serial"
  "errors"
)

type Config struct {
  Address          string
  MeasurementRange uint32
}

type Sensor interface {
  SetMeasurementRange( max uint32 ) error
  GetMeasurement()( uint32, error )
  Close()
}

type sensor struct {
  port *serial.Port
  cfg  Config
}

func NewSensor( c *Config )( Sensor, error ) {
  s := &sensor{}
  s.cfg = *c

  if s.cfg.MeasurementRange == 0 {
    s.cfg.MeasurementRange = 5000
  }

  sc := &serial.Config{ Name: s.cfg.Address, Baud: 9600 }
  port, err := serial.OpenPort( sc )
  if err != nil {
    return nil, err
  }

  s.port = port

  s.SetMeasurementRange( s.cfg.MeasurementRange )

  return s, nil
}

func ( s *sensor )SetMeasurementRange( max uint32 ) error {
  // 0xFF, 0x01, 0x99, 0x00, 0x00, 0x00, 0x13, 0x88, 0xCB
  var cmd = []byte { 0xff, 0x01, 0x99, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 }
  cmd[ 6 ] = byte( ( max & 0xff00 ) >> 8 )
  cmd[ 7 ] = byte( max & 0x00ff )
  cmd[ 8 ] = 0x00
  for i := 1; i < 8; i++ {
    cmd[ 8 ] = cmd[ 8 ] + cmd[ i ]
  }
  cmd[ 8 ] = 255 - cmd[ 8 ]
  cmd[ 8 ] = cmd[ 8 ] + 1

	if _, err := s.port.Write( cmd ); err != nil {
		return err
	}

  var resp = []byte { 0, 0, 0, 0, 0, 0, 0, 0, 0 }
  _, err := s.port.Read( resp )
  if err != nil {
    return err
  }

  var crc = byte( 0x00 )
  for i := 1; i < 8; i++ {
    crc += resp[ i ]
  }
  crc = 255 - crc
  crc += 1

  if resp[ 0 ] != 0xff || resp[ 1 ] != 0x99 || resp[ 8 ] != crc {
    return errors.New( "Bad response" )
  }

  return nil
}

func ( s *sensor )GetMeasurement()( uint32, error ) {
  // 0xFF,0x01,0x86,0x00,0x00,0x00,0x00,0x00,0x79
  var cmd = []byte { 0xff, 0x01, 0x86, 0x00, 0x00, 0x00, 0x00, 0x00, 0x79 }
  for i := 1; i < 8; i++ {
    cmd[ 8 ] = cmd[ 8 ] + cmd[ i ]
  }
  cmd[ 8 ] = 255 - cmd[ 8 ]
  cmd[ 8 ] = cmd[ 8 ] + 1

	if _, err := s.port.Write( cmd ); err != nil {
		return 0, err
	}

  var resp = []byte { 0, 0, 0, 0, 0, 0, 0, 0, 0 }
  _, err := s.port.Read( resp )
  if err != nil {
    return 0, err
  }

  var crc = byte( 0x00 )
  for i := 1; i < 8; i++ {
    crc += resp[ i ]
  }
  crc = 255 - crc
  crc += 1

  if resp[ 0 ] != 0xff || resp[ 1 ] != 0x99 || resp[ 8 ] != crc {
    return 0, errors.New( "Bad response" )
  }
  
  measure := ( uint32( resp[ 2 ] ) << 8 ) + uint32( resp[ 3 ] )

  return measure, nil
}

func ( s *sensor )Close() {
  s.port.Close()
}
