package mhz19b

import(
  "github.com/Necrys/serial"
  "errors"
  "log"
)

type Config struct {
  Address          string
  MeasurementRange uint32
  Debug            bool
  Autocalibration  bool
}

type Sensor interface {
  SetMeasurementRange( max uint32 ) error
  GetMeasurement()( uint32, error )
  Close()
  SetAutocalibration( bool ) error
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

  if s.cfg.Debug {
    log.Println( "NewSensor, config{ Address: \"%s\", MeasurementRange: %d, Debug: %v }",
      s.cfg.Address, s.cfg.MeasurementRange, s.cfg.Debug )
  }

  sc := &serial.Config{ Name: s.cfg.Address, Baud: 9600 }
  port, err := serial.OpenPort( sc )
  if err != nil {
    if s.cfg.Debug {
      log.Println( err )
    }
    return nil, err
  }

  s.port = port

  s.SetMeasurementRange( s.cfg.MeasurementRange )
  s.SetAutocalibration( s.cfg.Autocalibration )

  if s.cfg.Debug {
    log.Println( "Sensor successfully created" )
  }

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

  if s.cfg.Debug {
    log.Println( "SetMeasurementRange, cmd{ %x, %x, %x, %x, %x, %x, %x, %x, %x }",
      cmd[ 0 ], cmd[ 1 ], cmd[ 2 ], cmd[ 3 ], cmd[ 4 ], cmd[ 5 ], cmd[ 6 ], cmd[ 7 ], cmd[ 8 ] )
  }

	if _, err := s.port.Write( cmd ); err != nil {
		return err
	}

  var read = 0
  var resp = []byte { 0, 0, 0, 0, 0, 0, 0, 0, 0 }
  
  for read < 9 {
    n, err := s.port.Read( resp[ read: ] )
    if err != nil {
      return err
    }

    if s.cfg.Debug {
      log.Println( "SetMeasurementRange, read ", n, " bytes" )
    }
    
    read += n
  }

  if s.cfg.Debug {
    log.Println( "SetMeasurementRange, resp{ %x, %x, %x, %x, %x, %x, %x, %x, %x }",
      resp[ 0 ], resp[ 1 ], resp[ 2 ], resp[ 3 ], resp[ 4 ], resp[ 5 ], resp[ 6 ], resp[ 7 ], resp[ 8 ] )
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

  s.cfg.MeasurementRange = max

  return nil
}

func ( s *sensor )GetMeasurement()( uint32, error ) {
  // 0xFF,0x01,0x86,0x00,0x00,0x00,0x00,0x00,0x79
  var cmd = []byte { 0xff, 0x01, 0x86, 0x00, 0x00, 0x00, 0x00, 0x00, 0x79 }
  cmd[ 8 ] = 0
  for i := 1; i < 8; i++ {
    cmd[ 8 ] = cmd[ 8 ] + cmd[ i ]
  }
  cmd[ 8 ] = 255 - cmd[ 8 ]
  cmd[ 8 ] = cmd[ 8 ] + 1

  if s.cfg.Debug {
    log.Println( "GetMeasurement, cmd{ ",
      cmd[ 0 ], cmd[ 1 ], cmd[ 2 ], cmd[ 3 ], cmd[ 4 ], cmd[ 5 ], cmd[ 6 ], cmd[ 7 ], cmd[ 8 ], " }" )
  }

	if _, err := s.port.Write( cmd ); err != nil {
		return 0, err
	}

  var read = 0
  var resp = []byte { 0, 0, 0, 0, 0, 0, 0, 0, 0 }

  for read != 9 {
    n, err := s.port.Read( resp[ read: ] )
    if err != nil {
      return 0, err
    }

    if s.cfg.Debug {
      log.Println( "GetMeasurement, read ", n, " bytes" )
    }
    
    read += n
  }

  if s.cfg.Debug {
    log.Println( "GetMeasurement, resp{ ",
      resp[ 0 ], resp[ 1 ], resp[ 2 ], resp[ 3 ], resp[ 4 ], resp[ 5 ], resp[ 6 ], resp[ 7 ], resp[ 8 ], " }" )
  }

  var crc = byte( 0x00 )
  for i := 1; i < 8; i++ {
    crc += resp[ i ]
  }
  crc = 255 - crc
  crc += 1

  if resp[ 0 ] != 0xff || resp[ 1 ] != 0x86 || resp[ 8 ] != crc {
    return 0, errors.New( "Bad response" )
  }
  
  measure := ( uint32( resp[ 2 ] ) << 8 ) + uint32( resp[ 3 ] )

  return measure, nil
}

func ( s *sensor )SetAutocalibration( enable bool ) error {
  // 0xFF, 0x01, 0x79, 0xA0, 0x00, 0x00, 0x00, 0x00, 0x00
  var cmd = []byte { 0xff, 0x01, 0x79, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 }
  if enable {
    cmd[ 3 ] = 0xA0
  } else {
    cmd[ 3 ] = 0x00
  }

  for i := 1; i < 8; i++ {
    cmd[ 8 ] = cmd[ 8 ] + cmd[ i ]
  }
  cmd[ 8 ] = 255 - cmd[ 8 ]
  cmd[ 8 ] = cmd[ 8 ] + 1

  if s.cfg.Debug {
    log.Println( "SetAutocalibration, cmd{ ",
      cmd[ 0 ], cmd[ 1 ], cmd[ 2 ], cmd[ 3 ], cmd[ 4 ], cmd[ 5 ], cmd[ 6 ], cmd[ 7 ], cmd[ 8 ], " }" )
  }

	if _, err := s.port.Write( cmd ); err != nil {
		return err
	}

  var read = 0
  var resp = []byte { 0, 0, 0, 0, 0, 0, 0, 0, 0 }
  
  for read < 9 {
    n, err := s.port.Read( resp[ read: ] )
    if err != nil {
      return err
    }

    if s.cfg.Debug {
      log.Println( "SetAutocalibration, read ", n, " bytes" )
    }
    
    read += n
  }

  if s.cfg.Debug {
    log.Println( "SetAutocalibration, resp{ ",
      resp[ 0 ], resp[ 1 ], resp[ 2 ], resp[ 3 ], resp[ 4 ], resp[ 5 ], resp[ 6 ], resp[ 7 ], resp[ 8 ], " }" )
  }

  var crc = byte( 0x00 )
  for i := 1; i < 8; i++ {
    crc += resp[ i ]
  }
  crc = 255 - crc
  crc += 1

  if resp[ 0 ] != 0xff || resp[ 1 ] != 0x79 || resp[ 8 ] != crc {
    return errors.New( "Bad response" )
  }
  
  s.cfg.Autocalibration = enable

  return nil
}

func ( s *sensor )Close() {
  s.port.Close()
}
